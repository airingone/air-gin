package air_gin

import (
	airetcd "github.com/airingone/air-etcd"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/mitchellh/mapstructure"
	"testing"
)

func TestHttpServer(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化
	var addrs []string
	addrs = append(addrs, "127.0.0.1:2380")
	err := airetcd.RegisterLocalServerToEtcd(config.GetString("server.name"),
		config.GetUInt32("server.port"), addrs /*config.GetStringSlice("etcd.addrs")*/) //将服务注册到etcd集群
	if err != nil {
		log.Fatal("RegisterLocalServerToEtcd failed")
	}

	InitHttp("release") //上线使用release模式，支持debug，release，test，即为gin的模式
	httpRegister()
	RunHttp(":" + config.GetString("server.port"))
}

func httpRegister() {
	//无action接口，请求时不能有action字段
	RegisterServer("api/getuserinfo", PathNoAction, "POST", handleGetUserInfo) //注册请求路径为"api/test"的服务
	/*
		1.请求举例
		curl http://127.0.0.1:8080/api/getuserinfo  -d '{
		    "requestId": "123456789",
		    "requestMs": 1598848960000,
			"userId": "user123"
		}'

		2.response:
		成功：{"data":{"UserId":"user123","UserName":"user00","UserAge":20},"errCode":0,"errMsg":"succ","requestId":"123456789"}
		失败：{"data":null,"errCode":10001,"errMsg":"para err","requestId":"123456789"}
	*/

	//有action接口，请求时需要带action字段
	RegisterServer("api/userinfo", "mod", "POST", handleModUserInfo) //注册请求路径为"api/test"的服务
	/*
		1.请求举例
		curl http://127.0.0.1:8080/api/userinfo  -d '{
		    "requestId": "123456789",
		    "requestMs": 1598848960000,
			"action": "mod",
			"userId": "user123"
		}'
		2.response:
		成功：{"data":{"Userid":"user123"},"errCode":0,"errMsg":"succ","requestId":"123456789"}
		失败：{"data":null,"errCode":10001,"errMsg":"para err","requestId":"123456789"}
	*/
}

type GetUserInfoReq struct {
	BaseHeader `mapstructure:",squash"`
	UserId     string
}

type GetUserInfoRsp struct {
	UserId   string
	UserName string
	UserAge  int32
}

//业务逻辑-获取用户信息
func handleGetUserInfo(ctx *GinContext) {
	var req GetUserInfoReq
	err := mapstructure.Decode(ctx.Req, &req)
	if err != nil {
		ctx.SetErrMsg(10001, "para err")
		return
	}
	ctx.LogHandler.Info("req: %+v", req)
	if req.UserId == "" {
		ctx.SetErrMsg(10001, "para err")
		return
	}

	rsp := &GetUserInfoRsp{
		UserId:   req.UserId,
		UserName: "user00",
		UserAge:  20,
	}
	ctx.SetRsp(rsp)
}

type ModUserInfoReq struct {
	BaseHeader `mapstructure:",squash"`
	Action     string
	UserId     string
}

type ModUserInfoRsp struct {
	UserId string
}

//业务逻辑-修改用户信息
func handleModUserInfo(ctx *GinContext) {
	var req ModUserInfoReq
	err := mapstructure.Decode(ctx.Req, &req)
	if err != nil {
		ctx.SetErrMsg(10001, "para err")
		return
	}
	ctx.LogHandler.Info("req: %+v", req)
	if req.UserId == "" {
		ctx.SetErrMsg(10001, "para err")
		return
	}

	rsp := &ModUserInfoRsp{
		UserId: req.UserId,
	}
	ctx.SetRsp(rsp)
}
