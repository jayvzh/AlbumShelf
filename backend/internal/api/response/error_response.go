package response

// ErrorCode 对应 API.md §1 Error Code 枚举。
const (
	CodeInvalidPath     = "INVALID_PATH"
	CodeInvalidRequest  = "INVALID_REQUEST"
	CodeFileNotFound    = "FILE_NOT_FOUND"
	CodeFolderNotFound  = "FOLDER_NOT_FOUND"
	CodeUnsupported     = "UNSUPPORTED_FORMAT"
	CodeThumbnailFailed = "THUMBNAIL_FAILED"
	CodeInvalidRegex    = "INVALID_REGEX"
	CodeDatabaseError   = "DATABASE_ERROR"
	CodeInternalError   = "INTERNAL_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeConfigInvalid   = "CONFIG_INVALID"
)

// Error 统一错误对象（API.md §1：{ "error": { "code", "message" } }）。
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorBody 错误响应的外层包装。
type ErrorBody struct {
	Error Error `json:"error"`
}

// NewError 构造统一错误响应体。
func NewError(code, message string) ErrorBody {
	return ErrorBody{Error: Error{Code: code, Message: message}}
}
