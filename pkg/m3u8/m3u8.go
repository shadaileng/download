package m3u8

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"net/http"
	"github.com/shadaileng/download/pkg/download"
	"github.com/shadaileng/download/pkg/utils"
)

type Key struct {
	Method	string	`json: method`
	Uri		string	`json: uri`
}

type M3u8 struct {
	Url      string   `json: url`
	Uris     []map[string]string `json: uris`
	Keys     map[string]Key   `json: keys`
	Duration float64  `json: duration`
}

func (m3u8 *M3u8) String() string {
	return utils.Format("M3u8{Url: %v, Uris: %v, Keys: %v, Duration: %v}",
		m3u8.Url, m3u8.Uris, m3u8.Keys, m3u8.Duration)
}

func (m3u8 *M3u8) NewInfos(distPath string, httpClient *http.Client, status chan utils.Info) (map[string]utils.Info, error) {
	infos := map[string]utils.Info{}
	// 计算大小
	var wg = &sync.WaitGroup{}
	limit := make(chan int, 32)
	var mutex sync.Mutex
	for _, uri := range m3u8.Uris {
		wg.Add(1)
		go func(uri string) {
			defer wg.Done()
			if uri == "" {
				return
			}
			limit <- 1
			size, err := utils.Size(httpClient, uri)
			if err != nil {
				utils.Printf("Size(%v): %v", uri, err)
			}
			<- limit
			mutex.Lock()
			infos[uri] = utils.Info{
				Key:     uri,
				Url:     uri,
				Output:  utils.Outpath(distPath, uri),
				Start:   int64(0),
				Length:  size,
				DownLen: 0,
				Scale:   0.0,
				Status:  0,
				Retry:   5,
			}
			mutex.Unlock()
			status <- infos[uri]
		}(uri["url"])
	}
	wg.Wait()
	return infos, nil
}

func (m3u8 *M3u8) Download(distPath, socks5Url string, processFb, successFb, errorFb download.FBFunc) error {
	if m3u8.Url == "" {
		return errors.New("Url is empty")
	}
	m3u8.Keys = map[string]Key{}
	m3u8Path := utils.Outpath(distPath, m3u8.Url)
	err := m3u8.parse(m3u8Path, m3u8.Url, socks5Url, processFb, successFb, errorFb)
	if err != nil {
		return err
	}
	utils.Printf("m3u8: %v\n", m3u8)
	
	// if true {
	// 	return nil
	// }
	var wg = &sync.WaitGroup{}
	
	status := make(chan utils.Info)
	statusFilename := m3u8Path + ".json"
	go func() {
		defer wg.Done()
		wg.Add(1)
		download.DownloadStatus(status, m3u8Path)
	}()
	// utils.Printf("m3u8Path: %v\n", m3u8Path)
	httpClient, err := download.GenHttpClient(socks5Url)
	if err != nil {
		return err
	}
	infos, err := download.LoadInfos(statusFilename)
	if err != nil {
		infos, err = m3u8.NewInfos(distPath, httpClient, status)
		if err != nil {
			return err
		}
	}
	// utils.Printf("info: %v\n", infos)
	// wg.Wait()
	// if true {
	// 	return nil
	// }
	return download.DownloadAsyncResume(httpClient, status, infos, wg)
}

func (m3u8 *M3u8) parse(m3u8Path, m3u8Url, socks5Url string, processFb, successFb, errorFb download.FBFunc) error {
	m3u8PathBak := m3u8Path + ".bak"
	if !utils.Exists(m3u8PathBak) {
		err := download.Download(m3u8Url, m3u8PathBak, socks5Url, download.ReadBody, processFb, successFb, errorFb)
		if err != nil {
			return err
		}
	}
	
	err := utils.Copy(m3u8PathBak, m3u8Path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(m3u8Path, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	dataByte, err := utils.Loads(f)
	if err != nil {
		return err
	}
	stringData := string(dataByte)
	if !strings.HasPrefix(stringData, "#EXTM3U") {
		dataByte, err = utils.Base64Decode(stringData)
		if err != nil {
			return err
		}
		stringData = string(dataByte)
	}
	URL, _ := url.Parse(m3u8Url)
	lines := strings.Split(stringData, "\n")
	newContents := make([]string, 0)
	for _, line := range lines {
		content := strings.TrimSpace(line)
		if content == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-KEY") {
			key := Key{Method: "" , Uri: ""}
			for _, str := range strings.Split(line[10:], ",") {
				if strings.HasPrefix(str, "METHOD=") {
					key.Method = str[7:]
				}
				if strings.HasPrefix(str, "URI=") {
					key.Uri = utils.CleanPath(str[5:len(str)-1])
				}
			}
			if key.Uri != "" {				
				content = strings.Replace(line, key.Uri, utils.BasePath(key.Uri), 1)
				if !strings.HasPrefix(key.Uri, "http") {
					if strings.HasPrefix(key.Uri, "/") {
						key.Uri = utils.Format("%s://%s%s", URL.Scheme, URL.Host, key.Uri)
					} else {
						key.Uri = utils.Format("%s://%s%s/%s", URL.Scheme, URL.Host, utils.Dirname(URL.Path), key.Uri)
					}
				}
				m3u8.Uris = append(m3u8.Uris, map[string]string{"url": key.Uri, "path": utils.JoinPath(utils.Dirname(m3u8Path), utils.BasePath(key.Uri))})
			}
			m3u8.Keys[m3u8Url] = key
			newContents = append(newContents, content)
			continue
		}
		if strings.HasPrefix(line, "#EXTINF") {
			duration, _ := strconv.ParseFloat(strings.Split(line, ",")[0][8:], 64)
			m3u8.Duration += duration
			newContents = append(newContents, content)
			continue
		}
		if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
			newContents = append(newContents, content)
			break
		}
		if !strings.HasPrefix(line, "#") {
			if strings.Contains(line, ".m3u8") {
				if !strings.HasPrefix(line, "http") {
					if strings.HasPrefix(line, "/") {
						line = utils.CleanPath(line)
						line = utils.Format("%s://%s%s", URL.Scheme, URL.Host, line)
						if strings.HasPrefix(utils.CleanPath(content), utils.Dirname(URL.Path)) {
							content = utils.CleanPath(content)[len(utils.Dirname(URL.Path))+1:]
						} else {
							content = utils.CleanPath(content)[1:]
							// 保存路径 
							// utils.JoinPath(utils.Dirname(m3u8Path), content)
						}
					} else {
						line = utils.Format("%s://%s%s/%s", URL.Scheme, URL.Host, utils.Dirname(URL.Path), line)
					}
				} else {
					// 不同域名保存路径 
					// utils.JoinPath(utils.Dirname(m3u8Path), content)
					urlLine, _ := url.Parse(line)
					content = utils.Format("%s%s", urlLine.Host, urlLine.Path)
					if URL.Host == urlLine.Host {
						content = utils.CleanPath(urlLine.Path)
						if strings.HasPrefix(urlLine.Path, utils.Dirname(URL.Path)) {
							content = utils.CleanPath(urlLine.Path)[len(utils.Dirname(URL.Path))+1:]
						}
					}
				}
				err := m3u8.parse(utils.Outpath(utils.Dirname(m3u8Path), content), line, socks5Url, processFb, successFb, errorFb)
				if err != nil {
					return err
				}
				newContents = append(newContents, content)
				continue
			}

			if strings.HasPrefix(line, "http") {
				urlLine, _ := url.Parse(line)
				content = utils.Format("%s%s", urlLine.Host, urlLine.Path)
				Uri := map[string]string{"url": line, "path": utils.JoinPath(utils.Dirname(m3u8Path), content)}
				if URL.Host == urlLine.Host {
					content = utils.CleanPath(urlLine.Path)
					if strings.HasPrefix(urlLine.Path, utils.Dirname(URL.Path)) {
						content = utils.CleanPath(urlLine.Path)[len(utils.Dirname(URL.Path))+1:]
					}
				}
				Uri["path"] = utils.JoinPath(utils.Dirname(m3u8Path), content)
				m3u8.Uris = append(m3u8.Uris, Uri)
				newContents = append(newContents, content)
				continue
			}
			if strings.HasPrefix(line, "/") {
				// [scheme:][//[userinfo@]host][/]path[?query][#fragment]
				line = utils.Format("%s://%s%s", URL.Scheme, URL.Host, line)
				urlLine, _ := url.Parse(line)
				if strings.HasPrefix(utils.CleanPath(content), utils.Dirname(URL.Path)) {
					content = utils.CleanPath(urlLine.Path)[len(utils.Dirname(URL.Path))+1:]
				} else {
					content = utils.CleanPath(urlLine.Path)[1:]
				}
			} else {
				line = utils.Format("%s://%s%s/%s", URL.Scheme, URL.Host, utils.Dirname(URL.Path), line)
				urlLine, _ := url.Parse(line)
				content = utils.BasePath(urlLine.Path)
			}
			
			Uri := map[string]string{"url": line, "path": utils.JoinPath(utils.Dirname(m3u8Path), content)}
			m3u8.Uris = append(m3u8.Uris, Uri)
		}
		newContents = append(newContents, content)
	}
	// 修改内容
	// for _, content := range newContents {
	// 	println(content)
	// }
	utils.Dumps(f, []byte(strings.Join(newContents, "\n")))
	return nil
}
