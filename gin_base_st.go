package air_gin

//http json基础请求header，请求协议可以使用，也可以不使用
type BaseHeader struct {
	RequestId string //request id
	RequestMs int64  //发起请求时间，单位为毫秒
}
