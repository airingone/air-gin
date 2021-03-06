package air_gin

import (
	"encoding/json"
	"errors"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"io/ioutil"
	"net/http"
	"time"
)
//gin http服务封装

var eng *gin.Engine               //gin engine
var ginConfig config.ConfigServer //server配置，程序初始化获取一次

var handlerFuncs = make(map[string]HttpHandlerFunc) //路由表
var handlerNoAction = make(map[string]uint32)       //默认路由路径
const (
	PathNoAction = "-" //无路由的标示
)

type HttpHandlerFunc func(*GinContext) //逻辑执行函数定义

//初始化http
//mode: gin mode，默认release
func InitHttp(mode string) {
	ginConfig = config.GetServerConfig("server")

	if mode != gin.DebugMode && mode != gin.ReleaseMode &&
		mode != gin.TestMode {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	eng = gin.Default()
}

//启动http server
//addr: 监听地址，如":8080"
func RunHttp(addr string) {
	eng.Run(addr)
}

//一个请求的context
type GinContext struct {
	Ctx        *gin.Context           //context
	LogHandler *log.LogHandler        //log handler，用于打印requestid
	Config     *config.ConfigServer   //config
	RequestId  string                 //RequestId
	EnterMs    uint64                 //收到请求时时间戳
	Req        map[string]interface{} //接受数据
	Rsp        map[string]interface{} //返回数据
	ErrCode    uint32                 //错误码
	ErrMsg     string                 //错误信息
}

//创建
//ctx: gin context
func NewGinContext(ctx *gin.Context) *GinContext {
	ginCtx := &GinContext{
		Ctx:        ctx,
		LogHandler: log.NewLogHandler(),
		Config:     &ginConfig,
		Req:        make(map[string]interface{}),
		Rsp:        make(map[string]interface{}),
		ErrCode:    0,
		ErrMsg:     "succ",
	}

	return ginCtx
}

//设置log handler
//requestId: requestId
func (c *GinContext) SetLogHandler(requestId string) {
	c.RequestId = requestId
	c.LogHandler.SetRequestId(requestId)
}

//设置回包数据
//rsp: 回复包
func (c *GinContext) SetRsp(rsp interface{}) {
	c.Rsp["data"] = rsp
}

//获取当前毫秒时间
func (c *GinContext) GetCurrMs() uint64 {
	return uint64(time.Now().UnixNano() / 1e6)
}

//设置错误信息
//errCode: 错误码
//errMsg: 错误信息
func (c *GinContext) SetErrMsg(errCode uint32, errMsg string) {
	c.ErrCode = errCode
	c.ErrMsg = errMsg
}

//成功回包
func (c *GinContext) Response() {
	c.Ctx.JSON(http.StatusOK, gin.H{"requestId": c.RequestId, "errCode": c.ErrCode, "errMsg": c.ErrMsg, "data": c.Rsp["data"]})
	c.LogHandler.Info("Server: Response succ, rsp: %+v, errCode: %d, errMsg: %s", c.Rsp, c.ErrCode, c.ErrMsg)
}

//错误回包
//errCode: 错误码
//errMsg: 错误信息
func (c *GinContext) ResponseError(errCode uint32, errMsg string) {
	c.Ctx.JSON(http.StatusOK, gin.H{"requestId": c.RequestId, "errCode": errCode, "errMsg": errMsg, "data": c.Rsp["data"]})
	c.LogHandler.Info("Server: Response succ, rsp: %+v, errCode: %d, errMsg: %s", c.Rsp, c.ErrCode, c.ErrMsg)
}

//注册http请求，path为url路径，action为路由参数，为"-"则表示无路由
//path: http path
//action: request action
//methods: POST or GET
//handle: 业务逻辑处理函数
func RegisterServer(path string, action string, methods string, handle HttpHandlerFunc) error {
	if len(path) < 1 {
		return errors.New("Path err")
	}
	if path[0] != '/' {
		path = "/" + path
	}

	if len(action) < 1 {
		return errors.New("Action err")
	}
	if action == PathNoAction {
		handlerNoAction[path] = 1
	}

	handlerFuncs[path+action] = handle //注册handle处理函数

	if methods == "POST" {
		eng.POST(path, Server)
	} else if methods == "GET" {
		eng.GET(path, Server)
	} else if methods == "ALL" {
		eng.POST(path, Server)
		eng.GET(path, Server)
	}

	return nil
}

//http服务入口函数
//ctx: gin context
func Server(ctx *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	//初始化ctx
	c := NewGinContext(ctx)
	c.EnterMs = c.GetCurrMs()

	//读请求数据
	reqBytes, err := ioutil.ReadAll(c.Ctx.Request.Body)
	if err != nil {
		log.Info("[GIN]: Server Read http body data err, err: %+v", err)
		c.ResponseError(100, "Server:http read body err")
		return
	}
	err = json.Unmarshal(reqBytes, &c.Req)
	if err != nil {
		log.Error("[GIN]: Server Read http body data err, err: %+v", err)
		c.ResponseError(100, "Server:http body unmarshal err")
		return
	}
	log.Info("[GIN]: Server req: %+v", c.Req)

	//request id
	if _, ok := c.Req["requestId"].(string); !ok {
		log.Info("[GIN]: Server requestId not exist, req: %+v", c.Req)
		c.ResponseError(100, "Server:http body not have requestId err")
		return
	}
	c.SetLogHandler(c.Req["requestId"].(string))

	//网络耗时处理
	requestMs := cast.ToUint64(c.Req["requestMs"])
	if ginConfig.NetTimeOutMs != 0 {
		if c.EnterMs-requestMs > uint64(ginConfig.NetTimeOutMs) {
			c.LogHandler.Info("Server: requestMs timeout, req: %+v", c.Req)
			c.ResponseError(100, "Server:http request net time too bigger err")
			return
		}
	}

	//处理请求
	action := PathNoAction
	if _, ok := c.Req["action"].(string); ok {
		action = c.Req["action"].(string)
	}
	if action == PathNoAction {
		if _, ok := handlerNoAction[c.Ctx.FullPath()]; !ok {
			c.LogHandler.Info("Server: requestMs Action not support, req: %+v", c.Req)
			c.ResponseError(100, "Server:path not support")
			return
		}
	}
	if _, ok := handlerFuncs[c.Ctx.FullPath()+action]; !ok {
		c.LogHandler.Info("Server: requestMs Action not support, req: %+v", c.Req)
		c.ResponseError(100, "Server:action not support")
		return
	}
	handlerFuncs[c.Ctx.FullPath()+action](c) //执行业务逻辑

	//返回数据
	c.Response()

	//返回耗时监控 这里要加打点监控，需要有监控系统支持 todo
	return
}
