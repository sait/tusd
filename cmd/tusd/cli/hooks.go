package cli

import (
	"fmt"
	"strconv"

	"github.com/sait/tusd"
	"github.com/sait/tusd/cmd/tusd/cli/hooks"
)

var hookHandler hooks.HookHandler = nil

type hookDataStore struct {
	tusd.DataStore
}

func (store hookDataStore) NewUpload(info tusd.FileInfo) (id string, err error) {
	if output, err := invokeHookSync(hooks.HookPreCreate, info, true); err != nil {
		if hookErr, ok := err.(hooks.HookError); ok {
			return "", hooks.NewHookError(
				fmt.Errorf("pre-create hook failed: %s", err),
				hookErr.StatusCode(),
				hookErr.Body(),
			)
		}
		return "", fmt.Errorf("pre-create hook failed: %s\n%s", err, string(output))
	}
	return store.DataStore.NewUpload(info)
}

func SetupHookMetrics() {
	MetricsHookErrorsTotal.WithLabelValues(string(hooks.HookPostFinish)).Add(0)
	MetricsHookErrorsTotal.WithLabelValues(string(hooks.HookPostTerminate)).Add(0)
	MetricsHookErrorsTotal.WithLabelValues(string(hooks.HookPostReceive)).Add(0)
	MetricsHookErrorsTotal.WithLabelValues(string(hooks.HookPostCreate)).Add(0)
	MetricsHookErrorsTotal.WithLabelValues(string(hooks.HookPreCreate)).Add(0)
}

func SetupPreHooks(composer *tusd.StoreComposer) error {
	if Flags.FileHooksDir != "" {
		hookHandler = &hooks.FileHook{
			Directory: Flags.FileHooksDir,
		}
	} else if Flags.HttpHooksEndpoint != "" {
		hookHandler = &hooks.HttpHook{
			Endpoint:   Flags.HttpHooksEndpoint,
			MaxRetries: Flags.HttpHooksRetry,
			Backoff:    Flags.HttpHooksBackoff,
		}
	} else if Flags.PluginHookPath != "" {
		hookHandler = &hooks.PluginHook{
			Path: Flags.PluginHookPath,
		}
	} else {
		return nil
	}

	if err := hookHandler.Setup(); err != nil {
		return err
	}

	composer.UseCore(hookDataStore{
		DataStore: composer.Core,
	})

	return nil
}

func SetupPostHooks(handler *tusd.Handler) {
	go func() {
		for {
			select {
			case info := <-handler.CompleteUploads:
				invokeHookAsync(hooks.HookPostFinish, info)
			case info := <-handler.TerminatedUploads:
				invokeHookAsync(hooks.HookPostTerminate, info)
			case info := <-handler.UploadProgress:
				invokeHookAsync(hooks.HookPostReceive, info)
			case info := <-handler.CreatedUploads:
				invokeHookAsync(hooks.HookPostCreate, info)
			}
		}
	}()
}

func invokeHookAsync(typ hooks.HookType, info tusd.FileInfo) {
	go func() {
		// Error handling is taken care by the function.
		_, _ = invokeHookSync(typ, info, false)
	}()
}

func invokeHookSync(typ hooks.HookType, info tusd.FileInfo, captureOutput bool) ([]byte, error) {
	switch typ {
	case hooks.HookPostFinish:
		logEv(stdout, "UploadFinished", "id", info.ID, "size", strconv.FormatInt(info.Size, 10))
	case hooks.HookPostTerminate:
		logEv(stdout, "UploadTerminated", "id", info.ID)
	}

	if hookHandler == nil {
		return nil, nil
	}

	name := string(typ)
	logEv(stdout, "HookInvocationStart", "type", name, "id", info.ID)

	output, returnCode, err := hookHandler.InvokeHook(typ, info, captureOutput)

	if err != nil {
		logEv(stderr, "HookInvocationError", "type", string(typ), "id", info.ID, "error", err.Error())
		MetricsHookErrorsTotal.WithLabelValues(string(typ)).Add(1)
	} else {
		logEv(stdout, "HookInvocationFinish", "type", string(typ), "id", info.ID)
	}

	if typ == hooks.HookPostReceive && Flags.HooksStopUploadCode != 0 && Flags.HooksStopUploadCode == returnCode {
		logEv(stdout, "HookStopUpload", "id", info.ID)

		info.StopUpload()
	}

	return output, err
}
