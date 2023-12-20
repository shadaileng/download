package api

import (
	"path/filepath"
	"testing"
	_"fmt"
	"github.com/shadaileng/download/pkg/utils"
)

func _TestDownload(t *testing.T) {
	// url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	// url := "https://la3.killcovid2021.com/m3u8/907823/9078230.ts"
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	// filename := "X55udrAS.m3u8"
	// url := "https://ev-h.phncdn.com/hls/videos/202303/02/426518151/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_426518151.mp4.urlset/seg-298-f4-v1-a1.ts?validfrom=1702018236&validto=1702025436&ipa=198.244.144.187&hdl=-1&hash=0%2Fd3%2F%2FzOGt%2FiWddq6g7IMnW%2Bkfg%3D"
	// filename := "seg-298-f4-v1-a1.ts"
	url := "https://la3.killcovid2021.com/m3u8/907759/907759.m3u8"
	filename := filepath.Join("/config/workspace/project/go/src/github.com/shadaileng/download/dist", filepath.Base(url))
	socks5Url := "socks5://127.0.0.1:1080"
	// socks5Url := ""
	err := Download(url, filename, socks5Url, NewFileInfos)
	if err != nil {
		t.Error(`Download: `, err)
	}
}

func _TestDownloadM3u8(t *testing.T) {
	// url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	url := "https://la3.killcovid2021.com/m3u8/907759/907759.m3u8"
	filename := filepath.Join("/config/workspace/project/go/src/github.com/shadaileng/download/dist", 
		utils.ResourcePath(url))
	socks5Url := "socks5://127.0.0.1:1080"
	// socks5Url := ""
	err := Download(url, filename, socks5Url, NewM3u8Infos)
	if err != nil {
		t.Error(`Download: `, err)
	}
}


func TestParseM3u8(t *testing.T) {
    // url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
    // url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
    url := "https://vip3.lbbf9.com/20231129/9PyUSSFA/index.m3u8"	
    // url := "https://vip3.lbbf9.com/20231129/9PyUSSFA//700kb/hls/index.m3u8"	
    // url := "https://video56.zdavsp.com/video/20230613/6ab714fed9a9cb653d6eeec3937b70d6/index.m3u8"
    // url := "https://videozmwbf.0afaf5e.com/decry/vd/20231126/MDZhZmU0ND/151813/720/libx/hls/encrypt/index.m3u8"
	// url := "https://la3.killcovid2021.com/m3u8/907759/907759.m3u8"
	outpath := filepath.Join("/config/workspace/project/go/src/github.com/shadaileng/download/dist", 
		utils.ResourcePath(url))
	socks5Url := ""
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		t.Error(`GenHttpClient: `, err)
		return
	}
	uris, err := ParseM3u8(httpClient, url, outpath)
	if err != nil {
		t.Error(`ParseM3u8: `, err)
	}
	utils.Printf("uris: %v", uris)
}

func _TestNewM3u8Infos(t *testing.T) {
	url := "https://la3.killcovid2021.com/m3u8/907759/907759.m3u8"
    // url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
    // url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
    // url := "https://vip3.lbbf9.com/20231129/9PyUSSFA/index.m3u8"	
    // url := "https://vip3.lbbf9.com/20231129/9PyUSSFA//700kb/hls/index.m3u8"	
    // url := "https://video56.zdavsp.com/video/20230613/6ab714fed9a9cb653d6eeec3937b70d6/index.m3u8"
    // url := "https://videozmwbf.0afaf5e.com/decry/vd/20231126/MDZhZmU0ND/151813/720/libx/hls/encrypt/index.m3u8"
	outpath := filepath.Join("/config/workspace/project/go/src/github.com/shadaileng/download/dist", utils.ResourcePath(url))
	socks5Url := ""
	httpClient, err := GenHttpClient(socks5Url)
	if err != nil {
		t.Error(`GenHttpClient: `, err)
		return
	}
	status := make(chan *Info)
	go func() {
		for info := range status {
			utils.Printf("info: %v", info)
		}
	}()
	infos, err := NewM3u8Infos(httpClient, url, outpath, status)
	if err != nil {
		t.Error(`NewM3u8Infos: `, err)
	}
	utils.Printf("infos: %v", infos)
}