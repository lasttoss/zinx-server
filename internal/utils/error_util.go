package utils

import (
	"fmt"
	"github.com/aceld/zinx/zlog"
)

type ErrorCode int

const (
	InvalidRequestError ErrorCode = iota + 1000
	InvalidContextError
	ItemNotFoundError
	SystemError
	UnknownError
	UserIdContextError
	EncodeJsonError
	DecodeJsonError
	ContactAdminToSupportError
	JwtClaimsError
	AnotherDeviceLoginError
)

var errorMessages = map[ErrorCode]string{
	InvalidRequestError:        "invalid request",
	InvalidContextError:        "invalid context",
	ItemNotFoundError:          "item not found",
	SystemError:                "system error",
	UnknownError:               "unknown error",
	UserIdContextError:         "user id not found",
	EncodeJsonError:            "encode json error",
	DecodeJsonError:            "decode json error",
	ContactAdminToSupportError: "contact admin to support error",
	JwtClaimsError:             "jwt claims error",
	AnotherDeviceLoginError:    "another device login error",
}

type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewApiError(code ErrorCode) []byte {
	return []byte(fmt.Sprintf(`{"code": %d, "message": "%s"}`, code, errorMessages[code]))
}

func NewSystemError(msgId uint32) {
	zlog.Error("system error msgId: ", msgId)
}
