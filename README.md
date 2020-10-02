# http server组件
## 1.组件描述
http servers是基于gin封装的带action路由的http server组件。

## 2.如何使用
```
import (
    "github.com/airingone/config"
    "github.com/airingone/log"
    air_etcd "github.com/airingone/air-etcd"
    air_gin "github.com/airingone/air-gin"
)

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    air_etcd.RegisterLocalServerToEtcd(config.GetString("server.name"),
    	config.GetUInt32("server.port"), config.GetStringSlice("etcd.addrs")) //将服务注册到etcd集群,这里不是必须的，如果不用etcd发现服务则不需要
    
    air_gin.InitHttp("release") //上线使用release模式，支持debug，release，test，即为gin的模式
    httpRegister()              //注册服务函数
    air_gin.RunHttp(":" + config.GetString("server.port")) //启动服务
}

func httpRegister() {
    //无action接口，请求时不能有action字段
    air_gin.RegisterServer("api/getuserinfo", PathNoAction, "POST", handleGetUserInfo) //注册请求路径为"api/test"的服务
    //有action接口，请求时需要带action字段
    air_gin.RegisterServer("api/userinfo", "mod", "POST", handleModUserInfo) //注册请求路径为"api/test"的服务
}
```
handleGetUserInfo,handleModUserInfo函数请见[gin_test.go](https://github.com/airingone/air-gin/blob/master/gin_test.go)