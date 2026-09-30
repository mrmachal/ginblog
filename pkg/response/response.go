package response

// 业务响应码：与 HTTP status 分开维护，两层语义各自正确。
// 约定：
//
//	0       成功（Success 恒为 0）
//	1xxxx   客户端错误（参数、认证、限流）
//	2xxxx   业务错误（资源不存在、冲突、状态不允许）
//	5xxxx   服务端错误
//
// 前端主判断 `res.code === 0`，细分再用具体码；msg 仅用于展示，不要用于分支逻辑。
const (
	CodeSuccess = 0 // 成功

	// 1xxxx 客户端错误
	CodeInvalidParam   = 10001 // 参数校验失败
	CodeUnauthorized   = 10002 // 未登录或登录态失效
	CodeForbidden      = 10003 // 无权限
	CodeTooManyRequest = 10004 // 请求过于频繁

	// 2xxxx 业务错误
	CodeNotFound     = 20001 // 资源不存在
	CodeConflict     = 20002 // 资源冲突（如标题重复）
	CodeInvalidState = 20003 // 当前状态不允许该操作

	// 5xxxx 服务端错误
	CodeInternal    = 50000 // 服务器内部错误
	CodeDatabase    = 50001 // 数据库错误
	CodeUnavailable = 50002 // 服务暂不可用
)

// codeMsg 各响应码的默认文案，用于统一服务端提示；handler 可传自定义 msg 覆盖。
var codeMsg = map[int]string{
	CodeSuccess:        "success",
	CodeInvalidParam:   "参数校验失败",
	CodeUnauthorized:   "未登录或登录态已失效",
	CodeForbidden:      "没有权限执行该操作",
	CodeTooManyRequest: "请求过于频繁，请稍后重试",
	CodeNotFound:       "资源不存在",
	CodeConflict:       "资源冲突",
	CodeInvalidState:   "当前状态不允许该操作",
	CodeInternal:       "服务器内部错误",
	CodeDatabase:       "数据库错误",
	CodeUnavailable:    "服务暂不可用",
}

// MessageOf 返回响应码对应的默认文案，未知码返回通用错误提示。
func MessageOf(code int) string {
	if msg, ok := codeMsg[code]; ok {
		return msg
	}

	return "未知错误"
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type PageResponse struct {
	Total int64       `json:"total"`
	List  interface{} `json:"list"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

func NewResponse(code int, msg string, data interface{}) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}

func Success(data interface{}) *Response {
	return NewResponse(CodeSuccess, MessageOf(CodeSuccess), data)
}

func SuccessPage(total int64, list interface{}, page, size int) *Response {
	return NewResponse(CodeSuccess, MessageOf(CodeSuccess), &PageResponse{
		Total: total,
		List:  list,
		Page:  page,
		Size:  size,
	})
}

func Fail(code int, msg string) *Response {
	return NewResponse(code, msg, nil)
}

func FailWithData(code int, msg string, data interface{}) *Response {
	return NewResponse(code, msg, data)
}
