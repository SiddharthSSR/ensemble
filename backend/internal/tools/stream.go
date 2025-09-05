package tools

// TokenCallback is used to stream incremental text output.
type TokenCallback func(chunk string)

// ctx key for passing TokenCallback through context.
type ctxKey string

var CtxTokenCallbackKey ctxKey = "token_cb"

