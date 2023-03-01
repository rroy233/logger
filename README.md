### Logger

一个简单的日志模块



#### 功能

- [x] 多种输出方式(终端/文件)
- [x] 远程上报(需自行设计接收端)
- [x] 日志文件压缩归档
- [x] 对[gin](https://github.com/gin-gonic/gin)访问日志进行压缩归档



#### 配置使用

1. 引用本项目

   ```shell
   go get -u github.com/rroy233/logger
   ```

2. 初始化

   ```go
   import "github.com/rroy233/logger"
   //...
   logger.New(&logger.Config{
     StdOutput:      true,//是否输出到std_out
     StoreLocalFile: true,//是否输出到本地文件
     StoreRemote:    true,//是否启用远程上报
     RemoteConfig: logger.RemoteConfigStruct{
       RequestUrl: "http://127.0.0.1/api/logUpload",//汇报接收端的URL
       QueryKey:   "?key=xxx",//用于接收端的验证
     },//远程上报配置
     NotUseJson: true,//不使用json格式
   })
   ```

3. 使用

   ```go
   //共有日志等级：Debug、Info、Warn、Error、FATAL
   //所有日志等级均支持：Println、Fatalln、Printf、Fatalf方法
   //如
   logger.Info.Println("xxx")
   ```



#### 日志内容

将自动创建`./log/`目录，日志文件以日期命名，按天归档。

```
{"Caller":"./logger_test.go:21","level":"info","msg":"测试","time":"2023-11-04T05:01:04+08:00"}

INFO 2023-11-04 05:01:04 [./logger_test.go:21] 测试哈哈哈哈哈
```

gin日志(若有)需存储到`./log/gin.log`，若检测到该文件存在则会对gin日志文件按天归档。



#### 远程上报 - 接收端

请求体：

```go
//remoteReportReq 请求体结构
type remoteReportReq struct {
  Time int64  `json:"time" binding:"required"`//上报的时间戳(s)
	Data string `json:"data" binding:"required"`//上报的数据
}

//logData Data部分结构，同上日志格式示例json
type logData struct {
	Caller string    `json:"Caller"`//调用信息
	Level  string    `json:"level"`//日志等级
	Msg    string    `json:"msg"`//日志内容
	Time   time.Time `json:"time"`//记录时间
}
```

响应体：

```go
//remoteReportResp 响应体结构
type remoteReportResp struct {
	Status int    `json:"status"`//状态码
	Msg    string `json:"msg"`//提示文本
}

//Status状态码
const (
  logRemoteSuccess            = 0//成功
	logRemoteErrNotAuth         = -1001//验证失败
  logRemoteErrParamsInvalid   = -1002//参数无效
	logRemoteErrDataParseFailed = -1003//数据解析失败
)
```

