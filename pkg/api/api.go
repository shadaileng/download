package api

import (
	"io"
	"os"
	"net"
	"net/url"
	"net/http"
	"path/filepath"
	"golang.org/x/net/proxy"
	"crypto/tls"
	"encoding/json"
	"context"
	"time"
	"sync"
	"github.com/shadaileng/download/pkg/utils"
	"strings"
)
type NewInfoFunc func(*http.Client, string, string, chan *Info) (map[string]*Info, error)

func downloadTask(httpClient *http.Client, info *Info, status chan *Info, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	req, _ := http.NewRequest("GET", info.Url, nil)
	start, end := info.Start+info.DownLen, info.Start+info.Length
	if end > start {
		req.Header.Set("Range", utils.Format("bytes=%v-%v", start, end))
	}
	utils.Printf("[+][%v] get %v", info.Key, info.Url)
	resp, err := httpClient.Do(req)
	if err != nil {
		info.Status = -2
		info.Error = err.Error()
		utils.Printf("[-][%v] %v", info.Key, info.Error)
		status <- info
		return
	}
	if resp.StatusCode != 206 && resp.StatusCode != 200 {
		info.Status = -2
		info.Error = utils.Format("StatusCode: %v", resp.StatusCode)
		utils.Printf("[-][%v] %v", info.Key, info.Error)
		status <- info
		return
	}
	defer resp.Body.Close()
	if !utils.Exists(utils.Dirname(info.Output)) {
		err = os.MkdirAll(utils.Dirname(info.Output), 0777)
		if err != nil {
			utils.Printf("mkdir error: %v, %v\n", err, utils.Dirname(info.Output))
			return
		}
	}
	out, err := os.OpenFile(info.Output, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		info.Status = -2
		info.Error = err.Error()
		utils.Printf("[-][%v] %v", info.Key, info.Error)
		status <- info
		return
	}
	defer out.Close()
	out.Seek(start, 0)
	err = ReadBody(resp, out, func (data Info){
		// utils.Printf("status: %v\n", data)
		info.Length = data.Length
		info.DownLen = data.DownLen
		if data.Error != "" {
			info.Status = -1
			if info.Retry - 1 < 0 {
				info.Status = -2
			} else {
				info.Retry--
			}
			info.Error = data.Error
			utils.Printf("[-][%v] %v", info.Key, info.Error)
		}
		status <- info
	})
	if err != nil {
		info.Status = -1
		if info.Retry - 1 < 0 {
			info.Status = -2
		} else {
			info.Retry--
		}
		info.Error = err.Error()
		utils.Printf("[-][%v] %v", info.Key, info.Error)
		status <- info
		return
	}
	info.Status = 1
	utils.Printf("[-]task done: %v", info.Key)
	return
}

func TaskStatus(status chan *Info, outputPath string, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	statusFilename := outputPath + ".json"
	if !utils.Exists(utils.Dirname(outputPath)) {
		err := os.MkdirAll(utils.Dirname(outputPath), 0777)
		if err != nil {
			utils.Printf("mkdir error: %v, %v\n", err, utils.Dirname(outputPath))
			return
		}
	}
	f, err := os.OpenFile(statusFilename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		utils.Printf("Create status file error: %v, %v\n", err, statusFilename)
		return
	}
	defer f.Close()

	infos, err := LoadInfos(statusFilename)
	if err != nil {
		utils.Printf("LoadInfos: %v\n", err)
	}
	
		
	speed := &utils.Speed{NTotal: 0, Now: utils.Now()}
	downInfo := &utils.DownInfo{Output: outputPath}
	for info := range status {
		infos[info.Key] = info
		var downLen, size, chunks, avaible, Error int64
		flag := true
		for _, val := range infos {
			size += val.Length
			downLen += val.DownLen
			chunks++
			if val.Status == 0 || val.Status == -1{
				avaible++
				flag = false
			}
			if val.Status == -2{
				Error++
			}
		}
		if speed.NTotal == 0 {
			speed.NTotal = downLen
		}
		speed.Update(downLen)
		downInfo.Size = size
		downInfo.DSize = downLen
		downInfo.Chunks = chunks
		downInfo.Avaible = avaible
		downInfo.Speed = *speed
		utils.Printf("Download: %s\n", downInfo)
		data, _ := json.Marshal(infos)
		utils.Dumps(f, data)
		if flag {
			break
		}
	}
	utils.Printf("[+]Download Finished!")
}

func TaskProduce(infos map[string]*Info, tasks, status chan *Info, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	utils.Printf("task creat")
	for {
		flag := true
		for _, info := range infos {
			if info.Status == 0 || info.Status == -1 {
				flag = false
				if info.Status == -1 {
					info.Status = 0
					tasks <- info
				}
			}
		}
		if flag {
			break
		}
		time.Sleep(time.Second)
	}
	close(status)
	utils.Printf("task_creat exit")
}

func TaskComsume(httpClient *http.Client, infos map[string]*Info, tasks, status chan *Info, wg *sync.WaitGroup) {
	utils.Printf("task comsume")
	// limit := make(chan int, 64)
	for info_ := range tasks {
		info := info_
		// limit <- 1
		go downloadTask(httpClient, info, status, wg)
		// <- limit
	}
	utils.Printf("task_comsume exit")
}

func Download(url, outputPath, socks5Url string, newInfoFunc NewInfoFunc) error {
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		return err
	}
	status := make(chan *Info)
	tasks := make(chan *Info, 2)
	wg := &sync.WaitGroup{}
	// 打印协程
	go TaskStatus(status, outputPath, wg)
	// 创建请求协程
	infos, err := LoadInfos(outputPath + ".json")
	if err != nil {
		infos, err = newInfoFunc(httpClient, url, outputPath, status)
		if err != nil {
			return err
		}
		
	} else {
		for _, info := range infos {
			if info.Status == 0 || info.Status == -2{
				info.Status = -1
				info.Error = ""
				status <- info
            }
		}
	}

	go TaskProduce(infos, tasks, status, wg)
	// 消费请求协程
	go TaskComsume(httpClient, infos, tasks, status, wg)
	
	utils.Printf("Wait\n")
	wg.Wait()
	utils.Printf("end\n")
	return err
}

func LoadInfos(statusFilename string) (map[string]*Info, error) {
	infos := make(map[string]*Info)
	if !utils.Exists(statusFilename) {
		return infos, utils.Error("status file not exists")
	}
	statusFile, err := os.Open(statusFilename)
	if err != nil {
		return infos, err
	}
	defer statusFile.Close()
	buf, err := utils.Loads(statusFile)
	if err != nil {
		return infos, err
	}
	err = json.Unmarshal(buf, &infos)
	if err != nil {
		return infos, err
	}
	return infos, nil
}

func NewM3u8Infos(httpClient *http.Client, m3u8Url, outpath string, status chan *Info) (map[string]*Info, error) {
	urls, err := ParseM3u8(httpClient, m3u8Url, outpath)
	if err != nil {
		return nil, err
	}
	infos := make(map[string]*Info, 0)
	// 计算大小
	var wg = &sync.WaitGroup{}
	limit := make(chan int, 32)
	var mutex sync.Mutex
	for _, item := range urls {
		uri := item["uri"]
		path := item["path"]
		go func(uri, path string) {
			defer wg.Done()
			wg.Add(1)
			limit <- 1
			length, err := utils.Size(httpClient, uri)
			if err != nil {
				utils.Printf("Size(%v): %v", uri, err)
			}
			<- limit
			if length < 0 {
				length = 0
			}
			mutex.Lock()
			infos[uri] = &Info{
				Key:     uri,
				Url:     uri,
				Output:  path,
				Start:   int64(0),
				Length:  length,
				DownLen: 0,
				Scale:   0.0,
				Status:  -1,
				Retry:   5,
			}
			mutex.Unlock()
			status <- infos[uri]
		}(uri, path)
	}
	wg.Wait()
	return infos, nil	
}
func ParseM3u8(httpClient *http.Client, m3u8Url, outpath string) ([]map[string]string, error) {
	uris := make([]map[string]string, 0)
	m3u8PathBak := outpath + ".bak"
	if !utils.Exists(m3u8PathBak) {
		resp, err := httpClient.Get(m3u8Url)
		if err != nil {
			return uris, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return uris, utils.Error(utils.Format("status code %d", resp.StatusCode))
		}
		if !utils.Exists(utils.Dirname(m3u8PathBak)) {
			err := os.MkdirAll(utils.Dirname(m3u8PathBak), 0755)
			if err != nil {
				return uris, nil
			}
		}
		m3u8PathBakFile, err := os.Create(m3u8PathBak)
		if err != nil {
			return uris, err
		}
		defer m3u8PathBakFile.Close()
		_, err = io.Copy(m3u8PathBakFile, resp.Body)
		if err != nil {
			return uris, err
		}
	}
	err := utils.Copy(m3u8PathBak, outpath)
	if err != nil {
		return uris, err
	}
	outpathFile, err := os.OpenFile(outpath, os.O_RDWR, 0666)
	if err != nil {
		return uris, err
	}
	defer outpathFile.Close()

	dataByte, err := utils.Loads(outpathFile)
	if err != nil {
		return uris, err
	}
	stringData := string(dataByte)
	if !strings.HasPrefix(stringData, "#EXTM3U") {
		dataByte, err = utils.Base64Decode(stringData)
		if err != nil {
			return uris, err
		}
		stringData = string(dataByte)
	}
	lines := strings.Split(stringData, "\n")

	relatePath := utils.Dirname(outpath)
	URL, _ := url.Parse(m3u8Url)
	contens := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		content := line
		if strings.HasPrefix(line, "#EXT-X-KEY") {
			for _, str := range strings.Split(line[10:], ",") {
				if strings.HasPrefix(str, "URI=") {
					keyUri := utils.CleanPath(str[5:len(str)-1])
					content = strings.Replace(line, keyUri, utils.BasePath(keyUri), 1)
					if !strings.HasPrefix(keyUri, "http") {
						line = utils.CleanPath(line)
						if strings.HasPrefix(keyUri, "/") {
							keyUri = utils.Format("%s://%s%s", URL.Scheme, URL.Host, keyUri)
						} else {
							keyUri = utils.Format("%s://%s%s/%s", URL.Scheme, URL.Host, utils.Dirname(URL.Path), keyUri)
						}
					}
					uris = append(uris, map[string]string{"uri": keyUri, "path": utils.JoinPath(relatePath, utils.BasePath(keyUri))})
				}
			}
		}
		if strings.HasPrefix(line, "#EXTINF") {

		}
		if !strings.HasPrefix(line, "#") {
			lineUri := line
			if strings.HasPrefix(line, "http") {
				content = utils.ResourcePath(line)
				urlLine , _ := url.Parse(line)
				if URL.Host == urlLine.Host{
					content = urlLine.Path
					if strings.HasPrefix(urlLine.Path, utils.Dirname(URL.Path)) {
						content = urlLine.Path[len(utils.Dirname(URL.Path))+1:]
					}
				}
			} else {
				line = utils.CleanPath(line)
				lineUri = utils.Format("%s://%s%s/%s", URL.Scheme, URL.Host, utils.Dirname(URL.Path), line)
				if strings.HasPrefix(line, "/") {
					lineUri = utils.Format("%s://%s%s", URL.Scheme, URL.Host, line)
					content = line[1:]
					if strings.HasPrefix(line, utils.Dirname(URL.Path)) {
						content = line[len(utils.Dirname(URL.Path)) + 1:]
					}
				}
			}
			if strings.Contains(line, ".ts") {
				uris = append(uris, map[string]string{"uri": lineUri, "path": utils.JoinPath(relatePath, content)})
			}
			if strings.Contains(line, ".m3u8") {
				urisLine, err := ParseM3u8(httpClient, lineUri, utils.JoinPath(relatePath, content))
				if err == nil {
					uris = append(uris, urisLine...)
				}
			}
		}
		contens = append(contens, content)
		if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
			break
		}
	}
	utils.Dumps(outpathFile, []byte(strings.Join(contens, "\n")))
	return uris, nil
}

func NewFileInfos(httpClient *http.Client, url, outpath string, status chan *Info) (map[string]*Info, error) {
	resp, err := httpClient.Head(url)
	if err != nil {		
		return nil, err
	}
	defer resp.Body.Close()
	length := resp.ContentLength
	if length <= 0 {
		length = 0
	}
	if !utils.Exists(outpath) {
		err = os.MkdirAll(filepath.Dir(outpath), 0777)
		if err != nil {
			return nil, err
		}
		out, err := os.Create(outpath)
		if err != nil {
			return nil, err
		}
		defer out.Close()
		out.Truncate(length)
	}
	chunkSize := int64(6059)
	chunks := length / chunkSize
	if length % chunkSize != 0 {
		chunks++
	}
	infos := make(map[string]*Info, chunks)
	for index := int64(0); index < chunks; index++ {
		start := index * chunkSize
		end := (index+1)*chunkSize - 1
		if end > length {
			end = length
		}
		key := utils.Format("%v", index)
		infos[key] = &Info{
			Key:     utils.Format("%v", index),
			Url:     url,
			Output:  outpath,
			Start:   int64(start),
			Length:  int64(end - start + 1),
			DownLen: 0,
			Scale:   0.0,
			Status:  -1,
			Retry:   5,
		}
		status <- infos[key]
	}
	if chunks == 0 {
		infos[utils.Format("%v", 0)] = &Info{
			Key:     utils.Format("%v", 0),
			Url:     url,
			Output:  outpath,
			Start:   int64(0),
			Length:  int64(0),
			DownLen: 0,
			Scale:   0.0,
			Status:  -1,
			Retry:   5,
		}
		status <- infos[utils.Format("%v", 0)]
	}
	return infos, nil
}


func GenHttpClient(socks5Url string) (*http.Client, error) {
	// 创建一个自定义的 http.Client 并禁用证书验证
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if socks5Url != "" {
		dialer, err := proxyConn(socks5Url)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			c, e := dialer.Dial(network, addr)
			return c, e
		}
	}
	httpClient := &http.Client{
		Transport: transport,
	}
	return httpClient, nil
}

func proxyConn(socks5Url string) (proxy.Dialer, error) {
	// 设置 SOCKS5 代理地址
	proxyURL, err := url.Parse(socks5Url)
	if err != nil {
		return nil, err
	}

	// 创建一个 Dialer
	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return dialer, nil
}

type FBFunc func(Info)
// type ReadBodyParams struct {
// 	chunk_size int64 // = 1024 * 1024 * 10
// 	fsize      int64 // = resp.ContentLength
// 	buf              // = make([]byte, chunk_size)
// 	written    int64
// }

func ReadBody(resp *http.Response, out *os.File, processFb FBFunc) error {
	// defer trace()()
	var (
		chunk_size int64 = 1024 // * 1024 * 10
		fsize      int64 = resp.ContentLength
		buf              = make([]byte, chunk_size)
		written    int64
	)
	for {
		//读取bytes
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			//写入bytes
			nw, ew := out.Write(buf[0:nr])
			//数据长度大于0
			if nw > 0 {
				written += int64(nw)
			}
			//写入出错
			if ew != nil {
				if processFb != nil {
					processFb(Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("write error: %v", ew)})
				}
				return ew
			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				if processFb != nil {
					processFb(Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("write error: %v", io.ErrShortWrite)})
				}
				return io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				if processFb != nil {
					processFb(Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("read error: %v", er)})
				}
				return er
			}
			if processFb != nil {
				processFb(Info{Output: out.Name(), Length: fsize, DownLen: written})
			}
			break
		}
		//没有错误了快使用 callback
		if processFb != nil {
			processFb(Info{Output: out.Name(), Length: fsize, DownLen: written})
		}
	}
	return nil
}


type Info struct {
	Key     string  `json: key`
	Url     string  `json: url`
	Length  int64   `json: length`
	DownLen int64   `json: downLen`
	Scale   float64 `json: scale`
	Error   string   `json: error`
	Start   int64   `json: start`
	End     int64   `json: end`
	Status  int64   `json: status`
	Output  string  `json: output`
	Retry   int64   `json: retry`
}

func (info Info) String() string {
	return utils.Format("Info{Url: %v, Length: %v, downLen: %v, scale: %v, error: %v, start: %v, end: %v, status: %v, output: %v, Retry: %v}",
		info.Url, info.Length, info.DownLen, info.Scale, info.Error, info.Start, info.End, info.Status, info.Output, info.Retry)
}
