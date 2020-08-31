package air_gin

import (
	"encoding/json"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"io/ioutil"
	"net/http"
	"time"
)

var eng *gin.Engine               //gin engine
var ginConfig config.ConfigServer //server配置，程序初始化获取一次

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

func RunHttp(addr string) {
	eng.Run(addr)
}

type GinContext struct {
	Ctx        *gin.Context
	LogHandler *log.LogHandler
	Config     *config.ConfigServer
	RequestId  string
	EnterMs    uint64                 //收到请求时时间戳
	Req        map[string]interface{} //接受数据
	Rsp        map[string]interface{} //返回数据
	ErrCode    uint32                 //错误码
	ErrMsg     string                 //错误信息
}

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

func (c *GinContext) SetLogHandler(requestId string) {
	c.RequestId = requestId
	c.LogHandler.SetRequestId(requestId)
}

func (c *GinContext) SetRsp(rsp interface{}) {
	c.Rsp["data"] = rsp
}

func (c *GinContext) GetCurrMs() uint64 {
	return uint64(time.Now().UnixNano() / 1e6)
}

func (c *GinContext) SetErrMsg(errCode uint32, errMsg string) {
	c.ErrCode = errCode
	c.ErrMsg = errMsg
}

func (c *GinContext) Response() {
	c.Ctx.JSON(http.StatusOK, gin.H{"requestId": c.RequestId, "errCode": c.ErrCode, "errMsg": c.ErrMsg, "data": c.Rsp["data"]})
	c.LogHandler.Info("Server: Response succ, rsp: %+v, errCode: %d, errMsg: %s", c.Rsp, c.ErrCode, c.ErrMsg)
}

func (c *GinContext) ResponseError(errCode uint32, errMsg string) {
	c.Ctx.JSON(http.StatusOK, gin.H{"requestId": c.RequestId, "errCode": errCode, "errMsg": errMsg, "data": c.Rsp["data"]})
	c.LogHandler.Info("Server: Response succ, rsp: %+v, errCode: %d, errMsg: %s", c.Rsp, c.ErrCode, c.ErrMsg)
}

type HttpHandlerFunc func(*GinContext)

var handlerFuncs = make(map[string]HttpHandlerFunc)

//注册http请求，path为url路径，action为路由参数，为"-"则表示无路由
func RegisterServer(path string, action string, methods string, handle HttpHandlerFunc) {
	if len(path) < 1 {
		return
	}
	if path[0] != '/' {
		path = "/" + path
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
}

//http服务入口函数
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
		log.Info("Server: Read http body data err, err: %+v", err)
		c.ResponseError(100, "Server:http read body err")
		return
	}
	err = json.Unmarshal(reqBytes, &c.Req)
	if err != nil {
		log.Error("Server: Read http body data err, err: %+v", err)
		c.ResponseError(100, "Server:http body unmarshal err")
		return
	}
	log.Info("Server: req: %+v", c.Req)

	//request id
	if _, ok := c.Req["requestId"].(string); !ok {
		log.Info("Server: requestId not exist, req: %+v", c.Req)
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
	action := "-"
	if _, ok := c.Req["action"].(string); ok {
		action = c.Req["action"].(string)
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
