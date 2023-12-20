package main

import (
	"flag"
	_"os"
	_"strings"

	_"github.com/shadaileng/download/pkg/download"
	_"github.com/shadaileng/download/pkg/m3u8"
	"github.com/shadaileng/download/pkg/api"
	"github.com/shadaileng/download/pkg/utils"
	_"slices"
)

var mode = flag.String("mode", "file", "download mode: file|m3u8")
var output = flag.String("output", "./", "output path")
var proxy = flag.String("proxy", "", "proxy path")
var debug = flag.Bool("debug", false, "debug mode")
func main() {
	flag.Parse()
	if flag.Arg(0) == "" {
		utils.Printf("url is empty\n")
		return
	}
	if *mode != "file" && *mode != "m3u8" {
		utils.Printf("mode is invalid\n")
		return
	}
	if *output == "" {
		utils.Printf("output is empty\n")
		return
	}
	url := flag.Arg(0)
	// 下载文件
	if *mode == "file" {
		filename := utils.JoinPath(*output, utils.ResourceName(url))
		err := api.Download(url, filename, *proxy, api.NewFileInfos)
		if err != nil {
			utils.Printf("Failed to download: %v, %v\n", url, err)
		}
	}
	// 下载m3u8
	if *mode == "m3u8" {
		filename := utils.JoinPath(*output, utils.ResourcePath(url))
		err := api.Download(url, filename, *proxy, api.NewM3u8Infos)
		if err != nil {
			utils.Printf("Failed to download m3u8: %v, %v\n", url, err)
		}
	}
}