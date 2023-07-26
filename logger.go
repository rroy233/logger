package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/rroy233/logger.v2/targz"
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
	Warn  *logType
	Error *logType
	FATAL *logType
)

type logType struct {
	Level logrus.Level
}

type defaultFormatter struct {
}

type Config struct {
	//是否输出到终端
	StdOutput bool

	//是否输出到日志文件
	StoreLocalFile bool

	//是否启用远程汇报
	StoreRemote bool

	//远程汇报配置
	RemoteConfig RemoteConfigStruct

	//使用json格式
	UseJson bool

	//配置额外需要归档的日志文件
	//
	//归档操作会将文件当前内容复制到归档文件夹"/log/{name}-archives/"中按日期存档，并清空当前文件。
	ExtraArchive []ArchiveConfig
}

type ArchiveConfig struct {
	//日志名称
	Name string

	//日志文件位置
	File string
}

var GinLog = ArchiveConfig{"gin", "./log/gin.log"}

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
	defaultLogger.SetFormatter(new(defaultFormatter))
	defaultLogger.Level = logrus.DebugLevel

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
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(os.Stdout)
		} else {
			defaultLogger.SetOutput(os.Stdout)
			jsonLogger.SetOutput(io.Discard)
		}
	case 2:
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(logFile)
		} else {
			defaultLogger.SetOutput(logFile)
			jsonLogger.SetOutput(io.Discard)
		}
	case 4:
		defaultLogger.SetOutput(io.Discard)
		jsonLogger.SetOutput(remoteBuffer)
	case 3:
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
		} else {
			defaultLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
			jsonLogger.SetOutput(io.Discard)
		}
	case 5:
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, remoteBuffer))
		} else {
			defaultLogger.SetOutput(os.Stdout)
			jsonLogger.SetOutput(remoteBuffer)
		}
	case 6:
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(remoteBuffer, logFile))
		} else {
			defaultLogger.SetOutput(logFile)
			jsonLogger.SetOutput(remoteBuffer)
		}
	case 7:
		if config.UseJson {
			defaultLogger.SetOutput(io.Discard)
			jsonLogger.SetOutput(io.MultiWriter(os.Stdout, logFile, remoteBuffer))
		} else {
			defaultLogger.SetOutput(io.MultiWriter(os.Stdout, logFile))
			jsonLogger.SetOutput(remoteBuffer)
		}
	}

	//子logger
	Debug = &logType{Level: logrus.DebugLevel}
	Info = &logType{Level: logrus.InfoLevel}
	Warn = &logType{Level: logrus.WarnLevel}
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

	//检查额外归档的日志文件是否存在(后续进行日志分割)
	for _, archiveConfig := range config.ExtraArchive {
		archiveDir := fmt.Sprintf("./log/%s-archives/", archiveConfig.Name)
		if isExist(archiveConfig.File) == true && getFileSize(archiveConfig.File) > 0 {
			if isExist(archiveDir) == false {
				os.Mkdir(archiveDir, 0755)
				log.Println(archiveDir + " dir auto created! ")
			}

			fd, err := os.OpenFile(archiveConfig.File, os.O_RDWR, 0755)
			if err != nil {
				log.Fatalln("open", archiveConfig.File, "failed:", err)
			}
			defer fd.Close()

			yesterdayFileName := fmt.Sprintf("./log/%s-%s.log", time.Now().Add(-1*time.Hour).Format("2006-01-02"), archiveConfig.Name)
			if isExist(yesterdayFileName) == false {
				err = os.WriteFile(yesterdayFileName, nil, 0755)
				if err != nil {
					log.Fatalf("create %s.today.log failed:%s\n", archiveConfig.Name, err.Error())
				}
			}
			yestertodayFd, err := os.OpenFile(yesterdayFileName, os.O_APPEND|os.O_WRONLY, 0755)
			if err != nil {
				log.Fatalf("open %s.today.log failed:%s\n", archiveConfig.Name, err.Error())
			}
			defer yestertodayFd.Close()

			writer := bufio.NewWriter(yestertodayFd)
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

			//归零
			err = os.Truncate(archiveConfig.File, 0)
			if err != nil {
				log.Fatalf("%s log Truncate failed:%s\n", archiveConfig.Name, err.Error())
			}
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
		//处理额外的日志
		for _, archiveConfig := range config.ExtraArchive {
			if regexp.MustCompile(`\d{4}-\d{2}-\d{2}-`+archiveConfig.Name+`.log`).Match([]byte(entry.Name())) && entry.Name() != getTodayDateString()+"-"+archiveConfig.Name+".log" {
				err = targz.Create(fmt.Sprintf("./log/%s", entry.Name()), fmt.Sprintf("./log/%s-archives/%s.tar.gz", archiveConfig.Name, entry.Name()))
				if err != nil {
					log.Println("[archiveOldFile] targz.Create:", err)
				}
				_ = os.Remove(fmt.Sprintf("./log/%s", entry.Name()))
			}
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
