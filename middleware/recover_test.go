package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/theopenlane/echox"
)

func TestRecover(t *testing.T) {
	e := echox.New()
	buf := new(bytes.Buffer)
	e.Logger = &testLogger{output: buf}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := Recover()(func(c echox.Context) error {
		panic("test")
	})
	err := h(c)
	assert.Contains(t, err.Error(), "[PANIC RECOVER] test goroutine")
	assert.Equal(t, http.StatusOK, rec.Code) // status is still untouched. err is returned from middleware chain
	assert.Contains(t, buf.String(), "")     // nothing is logged
}

func TestRecover_skipper(t *testing.T) {
	e := echox.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	config := RecoverConfig{
		Skipper: func(c echox.Context) bool {
			return true
		},
	}
	h := RecoverWithConfig(config)(func(c echox.Context) error {
		panic("testPANIC")
	})

	var err error
	assert.Panics(t, func() {
		err = h(c)
	})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code) // status is still untouched. err is returned from middleware chain
}

func TestRecoverErrAbortHandler(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := Recover()(func(c echox.Context) error {
		panic(http.ErrAbortHandler)
	})

	defer func() {
		r := recover()
		if r == nil {
			assert.Fail(t, "expecting `http.ErrAbortHandler`, got `nil`")
		} else {
			if err, ok := r.(error); ok {
				assert.ErrorIs(t, err, http.ErrAbortHandler)
			} else {
				assert.Fail(t, "not of error type")
			}
		}
	}()

	hErr := h(c)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, hErr.Error(), "PANIC RECOVER")
}

func TestRecoverWithConfig(t *testing.T) {
	var testCases = []struct {
		name             string
		givenNoPanic     bool
		whenConfig       RecoverConfig
		expectErrContain string
		expectErr        string
	}{
		{
			name:             "ok, default config",
			whenConfig:       DefaultRecoverConfig,
			expectErrContain: "[PANIC RECOVER] testPANIC goroutine",
		},
		{
			name:             "ok, no panic",
			givenNoPanic:     true,
			whenConfig:       DefaultRecoverConfig,
			expectErrContain: "",
		},
		{
			name: "ok, DisablePrintStack",
			whenConfig: RecoverConfig{
				DisablePrintStack:   true,
				DisableErrorHandler: true,
			},
			expectErr: "testPANIC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			config := tc.whenConfig
			h := RecoverWithConfig(config)(func(c echox.Context) error {
				if tc.givenNoPanic {
					return nil
				}

				panic("testPANIC")
			})

			err := h(c)

			if tc.expectErrContain != "" {
				assert.Contains(t, err.Error(), tc.expectErrContain)
			} else if tc.expectErr != "" {
				assert.Contains(t, err.Error(), tc.expectErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, http.StatusOK, rec.Code) // status is still untouched. err is returned from middleware chain
		})
	}
}

func TestRecoverWithLogErrorFunc(t *testing.T) {
	var testCases = []struct {
		name             string
		whenConfig       RecoverConfig
		expectErrContain string
		expectNoErr      bool
	}{
		{
			name: "ok, LogErrorFunc receives stack and returns modified error",
			whenConfig: RecoverConfig{
				DisableErrorHandler: true,
				LogErrorFunc: func(c echox.Context, err error, stack []byte) error {
					assert.Equal(t, "testPANIC", err.Error())
					assert.Contains(t, string(stack), "goroutine")
					return echox.NewHTTPError(http.StatusInternalServerError, "handled panic")
				},
			},
			expectErrContain: "handled panic",
		},
		{
			name: "ok, LogErrorFunc returns nil suppresses error",
			whenConfig: RecoverConfig{
				DisableErrorHandler: true,
				LogErrorFunc: func(c echox.Context, err error, stack []byte) error {
					return nil
				},
			},
			expectNoErr: true,
		},
		{
			name: "ok, LogErrorFunc with DisablePrintStack receives nil stack",
			whenConfig: RecoverConfig{
				DisableErrorHandler: true,
				DisablePrintStack:   true,
				LogErrorFunc: func(c echox.Context, err error, stack []byte) error {
					assert.Nil(t, stack)
					return err
				},
			},
			expectErrContain: "testPANIC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := RecoverWithConfig(tc.whenConfig)(func(c echox.Context) error {
				panic("testPANIC")
			})

			err := h(c)

			if tc.expectNoErr {
				assert.NoError(t, err)
			} else if tc.expectErrContain != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContain)
			}

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRecoverWithDisableErrorHandler(t *testing.T) {
	e := echox.New()

	var errorHandlerCalled bool
	e.HTTPErrorHandler = func(c echox.Context, err error) {
		errorHandlerCalled = true
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	config := RecoverConfig{
		DisableErrorHandler: false,
	}

	h := RecoverWithConfig(config)(func(c echox.Context) error {
		panic("testPANIC")
	})

	err := h(c)

	assert.NoError(t, err) // error is handled by HTTPErrorHandler, not returned
	assert.True(t, errorHandlerCalled)
}
