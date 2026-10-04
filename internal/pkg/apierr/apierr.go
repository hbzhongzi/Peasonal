// Package apierr 定义统一的 HTTP 响应结构与错误码。
//
// 响应格式固定为两种之一：
//
//	{"data": ...}                                  // 成功
//	{"error": {"code": "not_found", "message": ...}} // 失败
//
// 为什么要有 code 而不只靠 HTTP 状态码：状态码不够表达业务语义
// （同样是 400，参数格式错误和参数值冲突前端要做不同处理），
// 而把 code 定义成字符串常量便于前端 switch，也便于日志聚合。
package apierr

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Code 是机器可读的错误标识。
type Code string

const (
	CodeInternal        Code = "internal_error"
	CodeInvalidArgument Code = "invalid_argument"
	CodeNotFound        Code = "not_found"
	CodeUnauthorized    Code = "unauthorized"
	CodeForbidden       Code = "forbidden"
	CodeConflict        Code = "conflict"
	CodePayloadTooLarge Code = "payload_too_large"
	CodeRateLimited     Code = "rate_limited"
	CodeUnavailable     Code = "unavailable"
)

// Error 是服务内部传递的错误。业务层返回它，传输层负责序列化。
type Error struct {
	Status  int    // HTTP 状态码
	Code    Code   // 机器可读错误码
	Message string // 给用户看的提示
	cause   error  // 内部原因，只进日志不外泄
}

func (e *Error) Error() string {
	if e.cause != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.cause.Error()
	}
	return string(e.Code) + ": " + e.Message
}

// Unwrap 让 errors.Is / errors.As 能穿透到根因。
func (e *Error) Unwrap() error { return e.cause }

// Cause 返回内部原因，供日志使用。
func (e *Error) Cause() error { return e.cause }

func New(status int, code Code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

func NotFound(msg string) *Error {
	return New(http.StatusNotFound, CodeNotFound, msg)
}

func InvalidArgument(msg string) *Error {
	return New(http.StatusBadRequest, CodeInvalidArgument, msg)
}

func Unauthorized(msg string) *Error {
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Forbidden(msg string) *Error {
	return New(http.StatusForbidden, CodeForbidden, msg)
}

func Conflict(msg string) *Error {
	return New(http.StatusConflict, CodeConflict, msg)
}

func PayloadTooLarge(msg string) *Error {
	return New(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, msg)
}

func RateLimited(msg string) *Error {
	return New(http.StatusTooManyRequests, CodeRateLimited, msg)
}

func Unavailable(msg string) *Error {
	return New(http.StatusServiceUnavailable, CodeUnavailable, msg)
}

func Internal(msg string, cause error) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: CodeInternal, Message: msg, cause: cause}
}

// FromError 把任意 error 转成 *Error。
//
// 未识别的错误一律按 500 处理，并且**只回通用文案** —— 内部错误细节不能出网。
func FromError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal("服务器内部错误", err)
}

// --- 响应写入 ---

type response struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

type errorBody struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

// WriteOK 写入成功响应。
func WriteOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Data: data})
}

// WriteCreated 写入 201 响应。
func WriteCreated(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, response{Data: data})
}

// WriteNoContent 写入 204 响应（无 body）。
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError 写入错误响应，并把内部原因记进日志。
func WriteError(w http.ResponseWriter, log *slog.Logger, err error) {
	e := FromError(err)
	if e.Status >= http.StatusInternalServerError && log != nil {
		log.Error("请求失败",
			slog.Int("status", e.Status),
			slog.String("code", string(e.Code)),
			slog.String("cause", errText(e.Cause())),
		)
	}
	writeJSON(w, e.Status, response{Error: &errorBody{Code: e.Code, Message: e.Message}})
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
