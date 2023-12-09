package download

import (
	_"path/filepath"
	"testing"
	_"fmt"
	"net/http"
	"os"
	"time"
)

func TestDownload(t *testing.T) {
	// url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	// url := "https://la3.killcovid2021.com/m3u8/907823/9078230.ts"
	// filename := filepath.Base(url)
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	// filename := "X55udrAS.m3u8"
	url := "https://ev-h.phncdn.com/hls/videos/202303/02/426518151/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_426518151.mp4.urlset/seg-298-f4-v1-a1.ts?validfrom=1702018236&validto=1702025436&ipa=198.244.144.187&hdl=-1&hash=0%2Fd3%2F%2FzOGt%2FiWddq6g7IMnW%2Bkfg%3D"
	filename := "seg-298-f4-v1-a1.ts"
	socks5Url := "socks5://127.0.0.1:1080"
	success := func(data utils.Info){
		utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", data.Output, data.Length, data.DownLen, float64(data.DownLen)/float64(data.Length)*100)
	}
	err := Download(url, filename, socks5Url, ReadBody, 
		success, success, nil)
	if err != nil {
		t.Error(`Download: `, err)
	}
}

func _TestString(t *testing.T) {
	var a, b uint64
	a, b = parseRange("0-1")
	utils.Printf("a: %d, b: %d\n", a, b)
	a, b = parseRange("1-")
	utils.Printf("a: %d, b: %d\n", a, b)
	a, b = parseRange("-1")
	utils.Printf("a: %d, b: %d\n", a, b)
}

func _TestDownloadAsync(t *testing.T) {
	url := "https://la3.killcovid2021.com/m3u8/907823/9078230.ts"
	filename := "9078230.ts"

	// url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	// filename := "907823.m3u8"
	
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	// filename := "X55udrAS.m3u8"
	socks5Url := "socks5://127.0.0.1:1080"
	err := DownloadAsync(url, filename + "_03", socks5Url, 6069)
	if err != nil {
		t.Error(`Download: `, err)
	}
}
func _TestLoadInfos(t *testing.T) {
	// url := "https://la3.killcovid2021.com/m3u8/907823/9078230.ts"
	filename := "9078230.ts"

	// url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	// filename := "907823.m3u8"
	
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	// filename := "X55udrAS.m3u8"
	// socks5Url := "socks5://127.0.0.1:1080"
	infos, err := LoadInfos(filename + "_03.json")
	if err != nil {
		t.Error(`LoadInfos: `, err)
	}
	utils.Printf("infos: %+v\n", infos)
}
func _TestDownloadRange(t *testing.T) {
	// url := "https://la3.killcovid2021.com/m3u8/907823/9078230.ts"
	url := "https://la3.killcovid2021.com/m3u8/907823/907823.m3u8"
	filename := "907823.m3u8"
	// url := "https://askzycdn.com/20231124/X55udrAS/2000kb/hls/index.m3u8"
	// filename := "X55udrAS.m3u8"
	socks5Url := "socks5://127.0.0.1:1080"
	err := DownloadRange(url, filename + "_03", socks5Url, "0-255", 
		func(resp *http.Response, outpath string) error {
			out, _ := os.Create(filename + "_03")
			ReadBody(resp, out, 
				func(data utils.Info){
					utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", data.Output, data.Length, data.DownLen, float64(data.DownLen)/float64(data.Length)*100)
				}, nil, nil)
			return nil
		})
	if err != nil {
		t.Error(`Download: `, err)
	}
	err = DownloadRange(url, filename + "_03", socks5Url, "256-", 
		func(resp *http.Response, outpath string) error {
			out, _ := os.OpenFile(filename + "_03", os.O_WRONLY|os.O_APPEND, 0666)
			ReadBody(resp, out, 
				func(data utils.Info){
					utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", data.Output, data.Length, data.DownLen, float64(data.DownLen)/float64(data.Length)*100)
				}, nil, nil)
			return nil
		})
	if err != nil {
		t.Error(`Download: `, err)
	}
}


// func _TestDownloadM3u8(t *testing.T) {
// 	// url := "https://la3.killcovid2021.com/m3u8/907759/907759.m3u8"
// 	// url := "https://videozmwbf.0afaf5e.com/decry/vd/20231126/MDZhZmU0ND/151813/720/libx/hls/encrypt/index.m3u8"
// 	// url := "https://vip3.lbbf9.com/20231129/9PyUSSFA/index.m3u8"
// 	// url := "https://vip3.lbbf9.com/20231129/9PyUSSFA//700kb/hls/index.m3u8"
// 	// url := "https://video56.zdavsp.com/video/20230613/6ab714fed9a9cb653d6eeec3937b70d6/index.m3u8"
// 	// url := "https://cv-h.phncdn.com/hls/videos/202208/20/414042451/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_414042451.mp4.urlset/index-f4-v1-a1.m3u8?d5wDqMdJ4aRXpaqDv_if06sGefeRKg4thtfKWwW6R4ebwSeKMi2cCtSkx8tKkkIU0pMPlEZpVlD94VknHddb7q-x2fJQQ4JFKCLXqNZCktLB9-Fxw1Xina37677T21g8BA8GE34vM33Pxyar2fYszZCShBJUPdQP_furANFi3ClQaCpo5zapCnjlM-ZReqp8rofBvANbZRdo"
// 	// url := "https://ev-h.phncdn.com/hls/videos/202303/02/426518151/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_426518151.mp4.urlset/master.m3u8?validfrom=1702018236&validto=1702025436&ipa=198.244.144.187&hdl=-1&hash=0%2Fd3%2F%2FzOGt%2FiWddq6g7IMnW%2Bkfg%3D"
// 	url := "https://cv-h.phncdn.com/hls/videos/202311/26/443659581/,1080P_4000K,720P_4000K,480P_2000K,240P_1000K,_443659581.mp4.urlset/index-f1-v1-a1.m3u8?-fHPSShf7kqO7nzeK7wvY7NdsYKvHVqKXUgY6TtwY6AOmkznkM6yMoU8Jz97xEI0nzVRp5l3Omsqh7u5JLd_5PZJH4yrtBNzrhfww3-wzrm6URGiNptYuGyLV15eHVxN6Boo41OEXEJNCMhGj8RAqlvl8spC3imn3c-6sHYDOvj-TTiR9Y19kpPmzi3IfwDZ53bgE6Xa5jh-"
// 	outputPath := "/config/workspace/project/nginx/data/m3u8"
// 	socks5Url := "socks5://127.0.0.1:1080"
// 	m3u8 := &M3u8{Url: url, Uris: []map[string]string{}, Keys: nil, Duration: 0}
// 	err := m3u8.Download(outputPath, socks5Url, 
// 		func(data utils.Info){
// 			utils.Printf("Download: %s, %d, %d\t%3.2f%%\n", data.Output, data.Length, data.DownLen, float64(data.DownLen)/float64(data.Length)*100)
// 			}, nil, nil)
// 	if err != nil {
// 		t.Error(`m3u8.Download: `, err)
// 	}
// }

func _TestSpeed(t *testing.T) {
	utils.Printf("duration: %f\n", float64(time.Now().UnixNano()) / 1e6)
	speed := &Speed{NTotal: 26602276, Now: Now()-1000}
	speed.Update(26618660)
	utils.Printf("speed: %s\n", speed)
}
