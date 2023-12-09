package download

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/net/proxy"
	"github.com/shadaileng/download/pkg/utils"
)

type FBFunc func(data utils.Info)
type SuccessFunc func(outputPath string, length, downLen int64)
type ErrorFunc func(outputPath string, length, downLen int64, err error)
type ReadBodyFunc func(resp *http.Response, out *os.File, processFb, successFb, errorFb FBFunc) error

func Download(url, outputPath, socks5Url string, readBody ReadBodyFunc, processFb, successFb, errorFb FBFunc) error {
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		return err
	}
	// 使用自定义的 httpClient 发起请求
	resp, err := httpClient.Get(url)
	// utils.Printf("resp: %v\n", resp.Body)
	// resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = os.MkdirAll(filepath.Dir(outputPath), 0777)
	if err != nil {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return readBody(resp, out, processFb, successFb, errorFb)
}

func LoadInfos(statusFilename string) (map[string]utils.Info, error) {
	infos := make(map[string]utils.Info)
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

func DownloadAsyncResume(httpClient *http.Client, status chan utils.Info, infos map[string]utils.Info, wg *sync.WaitGroup) error {
	// var wg = sync.WaitGroup{}
	limit := make(chan int, 64)
	// go DownloadAsyncResumeWork(httpClient, status, works, wg)
	flag := 1
	for key, info := range infos {
		if key == "status" {
			// status <- info
			continue
		}
		if info.Status == 1{
			continue
		}
		if info.Status == -1{
			info.Status = 0
			info.Retry = 5
		}
		wg.Add(1)
		flag++
		go func(httpClient *http.Client, status chan utils.Info, info utils.Info, wg *sync.WaitGroup) error {
			limit <- 1
			utils.Printf("key: %v, info: %v\n", key, info)
			DownloadAsyncResumeWork(httpClient, status, info, wg)
			<- limit
			return nil
		}(httpClient, status, info, wg)
	}
	if flag <= 1 {
		status <- infos["status"]
	}
	wg.Wait()
	return nil
}


func DownloadAsyncResumeWork(httpClient *http.Client, status chan utils.Info, info utils.Info, wg *sync.WaitGroup) error {
	defer wg.Done()
	key := info.Key
	req, _ := http.NewRequest("GET", info.Url, nil)
	start, end := info.Start+info.DownLen, info.Start+info.Length
	if end > start {
		req.Header.Set("Range", utils.Format("bytes=%v-%v", start, end))
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		utils.Printf("httpClient.Do(req): %v\n", err)
		status <- utils.Info{
			Key:     key,
			Url:     info.Url,
			Output:  info.Output,
			Start:   info.Start,
			Length:  info.Length,
			DownLen: info.DownLen,
			Scale:   info.Scale,
			Status:  -1,
			Error:   utils.Format("httpClient.Do(req): %v", err),
			Retry:	 0,
		}
		return err
	}
	defer resp.Body.Close()
	if !utils.Exists(filepath.Dir(info.Output)) {
		err = os.MkdirAll(filepath.Dir(info.Output), 0777)
		if err != nil {
			utils.Printf("MkdirAll: %v\n", err)
			status <- utils.Info{
				Key:     key,
				Url:     info.Url,
				Output:  info.Output,
				Start:   info.Start,
				Length:  info.Length,
				DownLen: info.DownLen,
				Scale:   info.Scale,
				Status:  -1,
				Error:   utils.Format("MkdirAll: %v", err),
				Retry:	 0,
			}
			return err
		}
	}
	out, err := os.OpenFile(info.Output, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		utils.Printf("OpenFile: %v\n", err)
		status <- utils.Info{
			Key:     key,
			Url:     info.Url,
			Output:  info.Output,
			Start:   info.Start,
			Length:  info.Length,
			DownLen: info.DownLen,
			Scale:   info.Scale,
			Status:  -1,
			Error:   utils.Format("OpenFile: %v", err),
			Retry:	 0,
		}
		return err
	}
	defer out.Close()
	if info.Length == 0 && resp.ContentLength > 0 {
		info.Length = resp.ContentLength
		info.End = resp.ContentLength
	}
	// utils.Printf("resp: %v\n", resp)
	out.Seek(start, 0)
	// utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", outputPath, info.Length, start, float64(info.DownLen)/float64(info.Length)*100)

	return ReadBody(resp, out, func(data utils.Info) {
		// 缓存进度
		// utils.Printf("Download[%v]: %s, %d, %d\t%3.2f%%\n", key, data.Output, info.Length, info.DownLen + data.DownLen, float64(info.DownLen + data.DownLen) / float64(info.Length)*100)
		status <- utils.Info{
			Key:     key,
			Url:     info.Url,
			Output:  info.Output,
			Start:   info.Start,
			Length:  info.Length,
			DownLen: info.DownLen + data.DownLen,
			Scale:   float64(info.DownLen+data.DownLen) / float64(info.Length),
			Status:  0,
			Retry:   info.Retry,
		}
	}, func(data utils.Info) {
		// utils.Printf("Download[%v]: %s, %d, %d\t%3.2f%%\n", key, data.Output, info.Length, info.DownLen + data.DownLen, float64(info.DownLen + data.DownLen) / float64(info.Length)*100)
		
		status <- utils.Info{
			Key:     key,
			Url:     info.Url,
			Output:  info.Output,
			Start:   info.Start,
			Length:  info.Length,
			DownLen: info.DownLen + data.DownLen,
			Scale:   float64(info.DownLen+data.DownLen) / float64(info.Length),
			Status:  1,
			Retry:   info.Retry,
		}
	}, func(data utils.Info) {
		if info.Retry-1 >= 0 {
			info.Status = -1
			info.Retry--
		}
		// info.DownLen += data.DownLen
		// info.Scale = float64(info.DownLen+data.DownLen) / float64(info.Length)
		// info.Error = data.Error
		// status <- info
		// utils.Printf("Download[%v]: %s, %d, %d\t%3.2f%%\n", key, data.Output, info.Length, info.DownLen + data.DownLen, float64(info.DownLen + data.DownLen) / float64(info.Length)*100)

		status <- utils.Info{
			Key:     key,
			Url:     info.Url,
			Output:  info.Output,
			Start:   info.Start,
			Length:  info.Length,
			DownLen: info.DownLen + data.DownLen,
			Scale:   float64(info.DownLen+data.DownLen) / float64(info.Length),
			Status:  info.Status,
			Error:   data.Error,
			Retry:	 info.Retry,
		}
	})
}

func DownloadStatus(status chan utils.Info, outputPath string) {
	// defer close(status)
	statusFilename := outputPath + ".json"
	f, err := os.OpenFile(statusFilename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		utils.Printf("Create status file error: %v, %v\n", err, statusFilename)
		return
	}
	defer f.Close()

	status_Filename := outputPath + ".status"
	status_Filename_f, err := os.OpenFile(status_Filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		utils.Printf("Create status file error: %v, %v\n", err, status_Filename)
		return
	}
	defer status_Filename_f.Close()
	// utils.Printf("LoadInfos: %v\n", err)

	infos, err := LoadInfos(statusFilename)
	if err != nil {
		utils.Printf("LoadInfos: %v\n", err)
	}
	
	speed := &utils.Speed{NTotal: 0, Now: utils.Now()}
	downInfo := &utils.DownInfo{Output: outputPath}
	for stat := range status {
		infos[stat.Key] = stat
		// utils.Printf("Download: %s\n", stat)
		var downLen, size, chunks, avaible int64
		flag := true
		for key, val := range infos {
			if key == "status" {
				continue
			}
			size += val.Length
			downLen += val.DownLen
			chunks++
			if val.Status == 0 {
				avaible++
				flag = false
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
		if flag {
			tmp := infos["status"]
			tmp.Status = 1
			infos["status"] = tmp
			data, _ := json.Marshal(infos)
			utils.Dumps(f, data)
			dataDownInfo, _ := json.Marshal(downInfo)
			utils.Dumps(status_Filename_f, dataDownInfo)
			break
		}
		data, _ := json.Marshal(infos)
		utils.Dumps(f, data)
		dataDownInfo, _ := json.Marshal(downInfo)
		utils.Dumps(status_Filename_f, dataDownInfo)

		infos, err = LoadInfos(statusFilename)
		if err != nil {
			utils.Printf("LoadInfos: %v\n", err)
			continue
		}
	}
}

func NewInfos(url, outputPath string, size, chunkSize int64, status chan utils.Info) (map[string]utils.Info, error) {
	infos := map[string]utils.Info{}
	out, err := os.Create(outputPath)
	if err != nil {
		return infos, err
	}
	out.Truncate(size)
	out.Close()
	chunks := int64(size / int64(chunkSize))
	if size%int64(chunkSize) != 0 {
		chunks += 1
	}
	utils.Printf("chunks: %v\n", chunks)

	for index := int64(0); index < chunks; index++ {
		start := index * chunkSize
		end := (index+1)*chunkSize - 1
		if end > size {
			end = size
		}
		key := utils.Format("%v", index)
		infos[key] = utils.Info{
			Key:     utils.Format("%v", index),
			Url:     url,
			Output:  outputPath,
			Start:   int64(start),
			Length:  int64(end - start),
			DownLen: 0,
			Scale:   0.0,
			Status:  0,
			Retry:   5,
		}
		status <- infos[key]
	}
	infos["status"] = utils.Info{
		Key:    "status",
		Status: 0,
	}
	status <- infos["status"]
	return infos, nil
}

func DownloadAsync(url, outputPath, socks5Url string, chunkSize int64) error {
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		return err
	}

	var wg = &sync.WaitGroup{}
	status := make(chan utils.Info)
	statusFilename := outputPath + ".json"
	go func() {
		defer wg.Done()
		wg.Add(1)
		DownloadStatus(status, outputPath)
	}()
	infos, err := LoadInfos(statusFilename)
	if err != nil {
		// 新下载
		resp, err := httpClient.Head(url)
		if err != nil {
			return err
		}
		size := resp.ContentLength
		resp.Body.Close()
		if size <= 0 {
			// 不可续传
			return Download(url, outputPath, socks5Url, ReadBody, func(data utils.Info) {}, func(data utils.Info) {}, func(data utils.Info) {})
		}

		infos, err = NewInfos(url, outputPath, size, chunkSize, status)
		if err != nil {
			return err
		}
	}
	return DownloadAsyncResume(httpClient, status, infos, wg)
}

func DownloadRange(url, outputPath, socks5Url, Range string, readBody func(resp *http.Response, out string) error) error {
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("GET", url, nil)
	// req.Header.Set("User-Agent", "My-User-Agent")
	if Range != "" {
		req.Header.Set("Range", utils.Format("bytes=%s", Range))
	}

	// 使用自定义的 httpClient 发起请求
	resp, err := httpClient.Do(req)
	// utils.Printf("resp.Header: %v\n", resp.Header)
	// resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// beg, end := parseRange(Range)
	return readBody(resp, outputPath)
}

func ReadBody(resp *http.Response, out *os.File, processFb, successFb, errorFb FBFunc) error {
	// defer trace()()
	var (
		chunk_size int64 = 1024 * 1024 * 10
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
				if errorFb != nil {
					errorFb(utils.Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("write error: %v", ew)})
				}
				return ew
			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				if errorFb != nil {
					errorFb(utils.Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("write error: %v", io.ErrShortWrite)})
				}
				return io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				if errorFb != nil {
					errorFb(utils.Info{Output: out.Name(), Length: fsize, DownLen: written, Error: utils.Format("read error: %v", er)})
				}
				return er
			}
			if successFb != nil {
				successFb(utils.Info{Output: out.Name(), Length: fsize, DownLen: written})
			}
			break
		}
		//没有错误了快使用 callback
		if processFb != nil {
			processFb(utils.Info{Output: out.Name(), Length: fsize, DownLen: written})
		}
	}
	return nil
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
