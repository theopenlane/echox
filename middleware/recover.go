package middleware

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/theopenlane/echox"
)

// LogErrorFunc defines a function for custom logging in the middleware.
// It receives the context, recovered error, and stack trace as separate parameters.
// If this function returns nil, the centralized HTTPErrorHandler will not be called.
type LogErrorFunc func(c echox.Context, err error, stack []byte) error

// RecoverConfig defines the config for Recover middleware.
type RecoverConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper Skipper

	// Size of the stack to be printed.
	// Optional. Default value 4KB.
	StackSize int

	// DisableStackAll disables formatting stack traces of all other goroutines
	// into buffer after the trace for the current goroutine.
	// Optional. Default value false.
	DisableStackAll bool

	// DisablePrintStack disables printing stack trace.
	// Optional. Default value as false.
	DisablePrintStack bool

	// LogErrorFunc defines a function for custom logging in the middleware.
	// If set, this function handles logging instead of the default behavior
	// which embeds the stack trace in the error message.
	// If this function returns nil, the centralized HTTPErrorHandler will not be called.
	LogErrorFunc LogErrorFunc

	// DisableErrorHandler disables the call to centralized HTTPErrorHandler.
	// The recovered error is then passed back to upstream middleware, instead of swallowing the error.
	// Optional. Default value false.
	DisableErrorHandler bool
}

// DefaultRecoverConfig is the default Recover middleware config.
var DefaultRecoverConfig = RecoverConfig{
	Skipper:             DefaultSkipper,
	StackSize:           4 << 10, // 4 KB
	DisableStackAll:     false,
	DisablePrintStack:   false,
	LogErrorFunc:        nil,
	DisableErrorHandler: true,
}

// Recover returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() echox.MiddlewareFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a Recovery middleware with config or panics on invalid configuration.
func RecoverWithConfig(config RecoverConfig) echox.MiddlewareFunc {
	return toMiddlewareOrPanic(config)
}

// ToMiddleware converts RecoverConfig to middleware or returns an error for invalid configuration
func (config RecoverConfig) ToMiddleware() (echox.MiddlewareFunc, error) {
	if config.Skipper == nil {
		config.Skipper = DefaultRecoverConfig.Skipper
	}

	if config.StackSize == 0 {
		config.StackSize = DefaultRecoverConfig.StackSize
	}

	return func(next echox.HandlerFunc) echox.HandlerFunc {
		return func(c echox.Context) (returnErr error) {
			if config.Skipper(c) {
				return next(c)
			}

			defer func() {
				if r := recover(); r != nil {
					if r == http.ErrAbortHandler {
						panic(r)
					}

					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					var stack []byte
					if !config.DisablePrintStack {
						stack = make([]byte, config.StackSize)
						length := runtime.Stack(stack, !config.DisableStackAll)
						stack = stack[:length]
					}

					if config.LogErrorFunc != nil {
						err = config.LogErrorFunc(c, err, stack)
					} else if !config.DisablePrintStack {
						err = fmt.Errorf("[PANIC RECOVER] %w %s", err, stack)
					}

					if err != nil && !config.DisableErrorHandler {
						c.Error(err)
					} else {
						returnErr = err
					}
				}
			}()

			return next(c)
		}
	}, nil
}
