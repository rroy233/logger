package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rroy233/logger/targz"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

var (
	Debug *logType
	Info  *logType
	Error *logType
	FATAL *logType
)

type logType struct {
	Level logrus.Level
}

type Config struct {
	StdOutput      bool
	StoreLocalFile bool
	StoreRemote    bool
	RemoteConfig   RemoteConfigStruct
	NotUseJson     bool
}

type remoteReportReq struct {
	Time int64  `json:"time"`
	Data string `json:"data"`
}

type remoteReportResp struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type RemoteConfigStruct struct {
	RequestUrl string
	QueryKey   string

	//RemoteReporterNum 远程报告协程数，默认为3
	RemoteReporterNum int
}

const (
	logRemoteErrNotAuth         = -1001
	logRemoteErrParamsInvalid   = -1002
	logRemoteErrDataParseFailed = -1003
)

var nextDate string
var nowDate string
var nextDateUnix int64
var waitTime time.Duration
var firstRun bool

var jsonLogger *logrus.Logger
var defaultLogger *logrus.Logger
var logFile *os.File
var config *Config
var remoteBuffer *buffer

var remoteReporterNum int

// New 主程序启动时需要调用这个函数来初始化
func New(cf *Config) {
	config = cf
	nextDate = ""
	nextDateUnix = 0
	firstRun = true

	//初始化
	initLogger()

	//启动本地日志文件自动命名协程
	if config.StoreLocalFile {
		go localFileRenameWorker()
	}

	//启动远程汇报协程
	if config.StoreRemote {
		if config.RemoteConfig.RemoteReporterNum == 0 {
			remoteReporterNum = 3
		} else {
			remoteReporterNum = config.RemoteConfig.RemoteReporterNum
		}
		for i := 0; i < remoteReporterNum; i++ {
			go remoteReporter()
		}
	}
	return
}

func remoteReporter() {
	for true {
		out := remoteBuffer.GetOne()
		data := new(remoteReportReq)
		data.Time = time.Now().Unix()
		data.Data = out
		payloadBytes, err := json.Marshal(data)
		if err != nil {
			log.Println("[remoteReporter]json格式化失败：", err)
			continue
		}
		body := bytes.NewReader(payloadBytes)

		req, err := http.NewRequest("POST", config.RemoteConfig.RequestUrl+config.RemoteConfig.QueryKey, body)
		if err != nil {
			log.Println("[remoteReporter]NewRequest失败：", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		httpClient := http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println("[remoteReporter]http.DefaultClient.Do失败：", err)
			continue
		}
		if resp.StatusCode != 200 { //非200，重发
			log.Println("[remoteReporter]远端http错误：", resp)
			_, _ = remoteBuffer.Write([]byte(out + "\n"))
			time.Sleep(5 * time.Second)
			continue
		}

		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("[remoteReporter]Read body失败：", err)
			continue
		}
		resp.Body.Close()
		res := new(remoteReportResp)
		err = json.Unmarshal(respData, res)
		if err != nil {
			log.Println("[remoteReporter]body json解析失败：", err)
			continue
		}
		if res.Status != 0 { //汇报失败，准备重发
			if res.Status == logRemoteErrNotAuth { //客户端验证失败
				log.Fatalln("[remoteReporter]远端验证失败：", res.Msg)
			}
			log.Println("[remoteReporter]远端返回错误：", res.Msg)
			_, _ = remoteBuffer.Write([]byte(out + "\n"))
			time.Sleep(5 * time.Second)
		}
	}
}

// localFileRenameWorker 用于在后台运行的日志监控进程
func localFileRenameWorker() {
	for {
		if nextDate == "" || time.Now().Unix() >= nextDateUnix { //初次运行或已经过了下个日期
			t := time.Now()

			tm1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 5, 0, t.Location())
			tm2 := tm1.AddDate(0, 0, 1) //次日凌晨

			nextDate = tm2.Format("2006-01-02")
			nextDateUnix = tm2.Unix()

			waitTime = time.Until(tm2)

			log.Println("[系统服务][日志监控进程]" + "已确定下一个苏醒时间")

			time.Sleep(waitTime) //睡眠直至第二天凌晨醒来
		}
		_ = logFile.Close()
		initLogger()
	}
}

// initLogger 开启新的日志记录线程
func initLogger() {
	var err error

	if config.StoreLocalFile {
		nowDate = getTodayDateString()
		//日志输出文件
		if isExist("./log/") == false {
			os.Mkdir("./log/", 0755)
			log.Println("file dir auto created! ")
		}
		logFile, err = os.OpenFile("./log/"+nowDate+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Faild to open error logger file:", err)
		}
	}

	jsonLogger = logrus.New()
	jsonLogger.Formatter = &logrus.JSONFormatter{}
	jsonLogger.Level = logrus.DebugLevel

	defaultLogger = logrus.New()
	defaultLogger.Formatter = &logrus.TextFormatter{}

	//初始化remoteWriter
	if config.StoreRemote && firstRun == true {
		remoteBuffer = NewBuffer()
		firstRun = false
	}

	//计算输出方式
	index := 0
	if config.StdOutput {
		index += 1
	}
	if config.StoreLocalFile {
		index += 2
	}
	if config.StoreRemote {
		index += 4
	}
	switch index {
	case 1:
		if config.NotUseJson {
			defaultLogger.SetOutput(os.Stdout)
			jsonLogger.SetOutput(io.Discard)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(os.Stdout)
		}
	case 2:
		if config.NotUseJson {
			defaultLogger.SetOutput(logFile)
			jsonLogger.SetOutput(io.Discard)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(logFile)
		}
	case 4:
		defaultLogger.SetOutput(io.Discard)
		jsonLogger.SetOutput(remoteBuffer)
	case 3:
		if config.NotUseJson {
			defaultLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
			jsonLogger.SetOutput(io.Discard)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
		}
	case 5:
		if config.NotUseJson {
			defaultLogger.SetOutput(os.Stdout)
			jsonLogger.SetOutput(remoteBuffer)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, remoteBuffer))
		}
	case 6:
		if config.NotUseJson {
			defaultLogger.SetOutput(logFile)
			jsonLogger.SetOutput(remoteBuffer)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(remoteBuffer, logFile))
		}
	case 7:
		if config.NotUseJson {
			defaultLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
			jsonLogger.SetOutput(remoteBuffer)
		} else {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, logFile, remoteBuffer))
		}
	}

	//子logger
	Debug = &logType{Level: logrus.DebugLevel}
	Info = &logType{Level: logrus.InfoLevel}
	Error = &logType{Level: logrus.ErrorLevel}
	FATAL = &logType{Level: logrus.FatalLevel}

	//整理归档历史日志
	archiveOldFile()

	return
}

// getTodayDateString 获取今日日期string
func getTodayDateString() string {
	return time.Now().Format("2006-01-02")
}

func archiveOldFile() {
	if config.StoreLocalFile == false {
		return
	}

	//扫描log文件夹
	dir, err := os.ReadDir("./log")
	if err != nil {
		log.Fatalln("[archiveOldFile] ReadDir:", err)
	}

	//检查archive文件夹
	if isExist("./log/archives/") == false {
		os.Mkdir("./log/archives/", 0755)
		log.Println("archives dir auto created! ")
	}

	//检查gin.log文件是否存在(后续进行日志分割)
	if isExist("./log/gin.log") == true && getFileSize("./log/gin.log") > 0 {
		if isExist("./log/gin-archives/") == false {
			os.Mkdir("./log/gin-archives/", 0755)
			log.Println("gin-archives dir auto created! ")
		}

		fd, err := os.OpenFile("./log/gin.log", os.O_RDWR, 0755)
		if err != nil {
			log.Fatalln("open gin.log failed:", err)
		}
		defer fd.Close()

		todayFileName := fmt.Sprintf("./log/%s-gin.log", time.Now().Format("2006-01-02"))
		if isExist(todayFileName) == false {
			err = os.WriteFile(todayFileName, nil, 0755)
			if err != nil {
				log.Fatalln("create gin.today.log failed:", err)
			}
		}
		todayFd, err := os.OpenFile(todayFileName, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			log.Fatalln("open gin.today.log failed:", err)
		}
		defer todayFd.Close()

		writer := bufio.NewWriter(todayFd)
		reader := bufio.NewReader(fd)
		for true {
			buf := make([]byte, 2048)
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			n, err = writer.Write(buf[:n])
			if err != nil {
				log.Fatalln("writer.Write error:", err)
			}
		}
		_, _ = writer.Write([]byte("\n"))
		err = writer.Flush()
		if err != nil {
			log.Fatalln("writer.Flush error:", err)
		}

		//gin.log归零
		err = os.Truncate("./log/gin.log", 0)
		if err != nil {
			log.Fatalln("gin log Truncate failed:", err)
		}
	}

	//压缩
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		//处理logger创建的日志
		if regexp.MustCompile(`\d{4}-\d{2}-\d{2}.log`).Match([]byte(entry.Name())) && entry.Name() != getTodayDateString()+".log" {
			err = targz.Create(fmt.Sprintf("./log/%s", entry.Name()), fmt.Sprintf("./log/archives/%s.tar.gz", entry.Name()))
			if err != nil {
				log.Println("[archiveOldFile] targz.Create:", err)
			}
			_ = os.Remove(fmt.Sprintf("./log/%s", entry.Name()))
		}
		//处理gin创建的日志
		if regexp.MustCompile(`\d{4}-\d{2}-\d{2}-gin.log`).Match([]byte(entry.Name())) && entry.Name() != getTodayDateString()+"-gin.log" {
			err = targz.Create(fmt.Sprintf("./log/%s", entry.Name()), fmt.Sprintf("./log/gin-archives/%s.tar.gz", entry.Name()))
			if err != nil {
				log.Println("[archiveOldFile] targz.Create:", err)
			}
			_ = os.Remove(fmt.Sprintf("./log/%s", entry.Name()))
		}
	}

	return
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func getFileSize(path string) int64 {
	fileStat, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return fileStat.Size()
}
