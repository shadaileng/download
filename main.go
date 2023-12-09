package main

import (
	"flag"
	"strings"

	"github.com/shadaileng/download/pkg/download"
	"github.com/shadaileng/download/pkg/m3u8"
	"github.com/shadaileng/download/pkg/utils"
)

var url  = flag.String("url", "", "download url")
var mode = flag.String("mode", "sync", "download mode: sync|async")
var output = flag.String("output", "./", "output path")
var proxy = flag.String("proxy", "", "proxy path")
func main() {
	flag.Parse()
	if *url == "" {
		utils.Printf("url is empty\n")
		return
	}
	if *mode != "sync" && *mode != "async" {
		utils.Printf("mode is invalid\n")
		return
	}
	if *output == "" {
		utils.Printf("output is empty\n")
		return
	}
	filename := utils.ResourceName(*url)
	success := func(data utils.Info){
		utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", data.Output, data.Length, data.DownLen, float64(data.DownLen)/float64(data.Length)*100)
	}
	if strings.Contains(*url, ".m3u8") {
		m3u8 := &m3u8.M3u8{Url: *url}
		err := m3u8.Download(*output, *proxy, success, success, nil)
		if err != nil {
			utils.Printf("Failed to download: %v, %v\n", url, err)
		}
	} else if *mode == "sync" {
		// 下载
		err := download.Download(*url, utils.JoinPath(*output, filename), *proxy, download.ReadBody, success, success, nil)
		if err != nil {
			utils.Printf("Failed to download: %v, %v\n", url, err)
		}
	} else if *mode == "async" {
		err := download.DownloadAsync(*url, utils.JoinPath(*output, filename), *proxy, 6069)
		if err != nil {
			utils.Printf("Failed to download: %v, %v\n", url, err)
		}
	}
}