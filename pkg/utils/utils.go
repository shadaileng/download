package utils

import (
	"fmt"
	"time"
	"net/url"
	"net/http"
	"path/filepath"
	"os"
	"io"
	"strconv"
	"strings"
	"encoding/json"
	"encoding/base64"
	"errors"
	"log"
	"runtime"
	"slices"
)


func trace() func() {
	start := time.Now()
	return func() {
		// fmt.Printf("duration: %s\n", time.Since(start))
		Printf("duration: %s\n", time.Since(start))
		// Println(Format("duration: %s", time.Since(start)))
	}
}

func Println(str string) {
	_, file, line, _ := runtime.Caller(1)
	log.Println(Format("%s:%d: ", file, line) + str)
}
func Printf(format string, args ...interface{}) {
	if !Exists("/config/workspace/project/go/src/github.com/shadaileng/download/log") {
		os.MkdirAll("/config/workspace/project/go/src/github.com/shadaileng/download/log", 0777)
	}
	logFile, err := os.OpenFile("/config/workspace/project/go/src/github.com/shadaileng/download/log/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	}
	logger := log.New(io.MultiWriter(os.Stdout, logFile), "Download: ", log.LstdFlags | log.Ldate | log.Lmicroseconds)
	if slices.Contains(os.Args[1:], "-debug") {
		// log.SetFlags(log.LstdFlags | log.Lshortfile | log.Ldate | log.Lmicroseconds)
		logger.SetPrefix("Debug: ")
		// logger.Printf(format, args...)
		_, file, line, _ := runtime.Caller(1)
		format = Format("%s:%d: ", file, line) + format
	}
	logger.Printf(format, args...)
}
func Format(formatStr string, args ...interface{}) string {
	return fmt.Sprintf(formatStr, args...)
}

func Outpath(distPath, Url string) string {
	u, _ := url.Parse(Url)
	return filepath.Join(distPath, Format("%s%s", u.Host, u.Path))
}
func Dirname(path string) string {
	return filepath.Dir(path)
}

func JoinPath(paths ...string) string {
	return filepath.Join(paths...)
}

func Ext(path string) string {
	return filepath.Ext(path)
}

func BasePath(path string) string {
	return filepath.Base(path)
}

func ResourceName(path string) string {
	return BasePath(ResourcePath(path))
}

func ResourcePath(path string) string {
	urlLine, _ := url.Parse(path)
	return Format("%s/%s", urlLine.Host, CleanPath(urlLine.Path))
}

func ResourceDir(path string) string {
	return Dirname(ResourcePath(path))
}

func CleanPath(path string) string {
	return filepath.Clean(path)
}

func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func Copy(srcPath, path string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func JsonLoad(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		Printf("jsonLoad err: %v\n", err)
	}
}


func JsonDump(i interface{}) string {
	data, _ := json.Marshal(i)
	return string(data)
}

func Loads(file *os.File) ([]byte, error) {
	result := []byte{}
	for {
		buf := make([]byte, 10240)
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		
		result = append(result, buf[:n]...)
	}
	return result, nil
}
func Dumps(file *os.File, data []byte) error {
	err := file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
    }
	return nil
}


func ParseRange(rangeStr string) (uint64, uint64) {
	if rangeStr == "" {
		return 0, 0
	}
	rangeStr = strings.TrimSpace(rangeStr)
	var start, end uint64
	if strings.Contains(rangeStr, "-") {
		// 1-2
		parts := strings.Split(rangeStr, "-")
		start, _ = strconv.ParseUint(parts[0], 10, 64)
		end, _ = strconv.ParseUint(parts[1], 10, 64)
	} else {
		// 1
		start, _ = strconv.ParseUint(rangeStr, 10, 64)
		end = 0
	}
	return start, end
}

func Error(err string) error {
	return errors.New(err)
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
	return Format("Info{Url: %v, Length: %v, downLen: %v, scale: %v, error: %v, start: %v, end: %v, status: %v, output: %v, Retry: %v}",
		info.Url, info.Length, info.DownLen, info.Scale, info.Error, info.Start, info.End, info.Status, info.Output, info.Retry)
}


func Now() float64 {
	return float64(time.Now().UnixNano()) / 1e6
}

type Speed struct {
	NTotal 	int64	`json: nTotal`
	LTotal 	int64	`lTotal`
	Now  	float64	`json: now`
	Last 	float64	`json: last`
	Val		string	`json: val`
}

func (s *Speed) Update(total int64) {
	s.LTotal = s.NTotal
	s.Last = s.Now
	
	if Now() - s.Last > 500 {
		s.NTotal = total
		s.Now = Now()
		bytes := float64(s.NTotal - s.LTotal) / (s.Now - s.Last) * float64(1000)
		// Printf("Now - Last: %.2f, bytes: %.2f\n", s.Now - s.Last, bytes)
		if float64(bytes) < float64(1024) {
			s.Val = Format("%.2fB/s", float64(bytes))
		} else if float64(bytes) / 1024.0 < float64(1024) {
			s.Val = Format("%.2fKB/s", float64(bytes) / 1024.0)
		} else if float64(bytes) / 1024.0 / 1024.0 < float64(1024) {
			s.Val = Format("%.2fMB/s", float64(bytes) / 1024.0 / 1024.0)
		} else {
			s.Val = Format("%.2fGB/s", float64(bytes) / 1024.0 / 1024.0 / 1024.0)
		}
	}
}

func (s *Speed) String() string {
	if s.Val == "" {
		return "0B/s"
	} 
	return s.Val
}

type DownInfo struct {
	Output	string	`json: output`
	Chunks	int64	`json: chunks`
	Avaible	int64	`json: avaible`
	Size	int64	`json: size`
	DSize	int64	`json: dsize`
	Speed	Speed	`json: speed`
}

func (d *DownInfo) String() string {
	persent := 0.0
	if d.Size > 0 {
		persent = float64(d.DSize) / float64(d.Size) * 100.0
	}
	return Format("Output: %s, Chunks: %d, Avaible: %d, Size: %d, DSize: %d, Speed: %s\t%.2f%%", d.Output, d.Chunks, d.Avaible, d.Size, d.DSize, d.Speed.Val, persent)
}

func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Size(httpClient *http.Client, url string) (int64, error) {
	resp, err := httpClient.Head(url)
	if err != nil {
		return 0, err
	}
	size := resp.ContentLength
	resp.Body.Close()
	return size, nil
}