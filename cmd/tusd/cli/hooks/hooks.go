package hooks

import (
	"github.com/sait/tusd"
)

type HookHandler interface {
	Setup() error
	InvokeHook(typ HookType, info tusd.FileInfo, captureOutput bool) ([]byte, int, error)
}

type HookType string

const (
	HookPostFinish    HookType = "post-finish"
	HookPostTerminate HookType = "post-terminate"
	HookPostReceive   HookType = "post-receive"
	HookPostCreate    HookType = "post-create"
	HookPreCreate     HookType = "pre-create"
)

type hookDataStore struct {
	tusd.DataStore
}

type HookError struct {
	error
	statusCode int
	body       []byte
}

func NewHookError(err error, statusCode int, body []byte) HookError {
	return HookError{err, statusCode, body}
}

func (herr HookError) StatusCode() int {
	return herr.statusCode
}

func (herr HookError) Body() []byte {
	return herr.body
}

func (herr HookError) Error() string {
	return herr.error.Error()
}
