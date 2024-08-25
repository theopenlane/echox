package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/theopenlane/echox"
)

func TestCSRF_tokenExtractors(t *testing.T) {
	var testCases = []struct {
		name                    string
		whenTokenLookup         string
		whenCookieName          string
		givenCSRFCookie         string
		givenMethod             string
		givenQueryTokens        map[string][]string
		givenFormTokens         map[string][]string
		givenHeaderTokens       map[string][]string
		expectError             string
		expectToMiddlewareError string
	}{
		{
			name:            "ok, multiple token lookups sources, succeeds on last one",
			whenTokenLookup: "header:X-CSRF-Token,form:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenHeaderTokens: map[string][]string{
				echox.HeaderXCSRFToken: {"invalid_token"},
			},
			givenFormTokens: map[string][]string{
				"csrf": {"token"},
			},
		},
		{
			name:            "ok, token from POST form",
			whenTokenLookup: "form:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenFormTokens: map[string][]string{
				"csrf": {"token"},
			},
		},
		{
			name:            "ok, token from POST form, second token passes",
			whenTokenLookup: "form:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenFormTokens: map[string][]string{
				"csrf": {"invalid", "token"},
			},
		},
		{
			name:            "nok, invalid token from POST form",
			whenTokenLookup: "form:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenFormTokens: map[string][]string{
				"csrf": {"invalid_token"},
			},
			expectError: "code=403, message=invalid csrf token",
		},
		{
			name:            "nok, missing token from POST form",
			whenTokenLookup: "form:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenFormTokens: map[string][]string{},
			expectError:     "code=400, message=Bad Request, internal=missing value in the form",
		},
		{
			name:            "ok, token from POST header",
			whenTokenLookup: "", // will use defaults
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenHeaderTokens: map[string][]string{
				echox.HeaderXCSRFToken: {"token"},
			},
		},
		{
			name:            "ok, token from POST header, second token passes",
			whenTokenLookup: "header:" + echox.HeaderXCSRFToken,
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenHeaderTokens: map[string][]string{
				echox.HeaderXCSRFToken: {"invalid", "token"},
			},
		},
		{
			name:            "nok, invalid token from POST header",
			whenTokenLookup: "header:" + echox.HeaderXCSRFToken,
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPost,
			givenHeaderTokens: map[string][]string{
				echox.HeaderXCSRFToken: {"invalid_token"},
			},
			expectError: "code=403, message=invalid csrf token",
		},
		{
			name:              "nok, missing token from POST header",
			whenTokenLookup:   "header:" + echox.HeaderXCSRFToken,
			givenCSRFCookie:   "token",
			givenMethod:       http.MethodPost,
			givenHeaderTokens: map[string][]string{},
			expectError:       "code=400, message=Bad Request, internal=missing value in request header",
		},
		{
			name:            "ok, token from PUT query param",
			whenTokenLookup: "query:csrf-param",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPut,
			givenQueryTokens: map[string][]string{
				"csrf-param": {"token"},
			},
		},
		{
			name:            "ok, token from PUT query form, second token passes",
			whenTokenLookup: "query:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPut,
			givenQueryTokens: map[string][]string{
				"csrf": {"invalid", "token"},
			},
		},
		{
			name:            "nok, invalid token from PUT query form",
			whenTokenLookup: "query:csrf",
			givenCSRFCookie: "token",
			givenMethod:     http.MethodPut,
			givenQueryTokens: map[string][]string{
				"csrf": {"invalid_token"},
			},
			expectError: "code=403, message=invalid csrf token",
		},
		{
			name:             "nok, missing token from PUT query form",
			whenTokenLookup:  "query:csrf",
			givenCSRFCookie:  "token",
			givenMethod:      http.MethodPut,
			givenQueryTokens: map[string][]string{},
			expectError:      "code=400, message=Bad Request, internal=missing value in the query string",
		},
		{
			name:                    "nok, invalid TokenLookup",
			whenTokenLookup:         "q",
			givenCSRFCookie:         "token",
			givenMethod:             http.MethodPut,
			givenQueryTokens:        map[string][]string{},
			expectToMiddlewareError: "extractor source for lookup could not be split into needed parts: q",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()

			q := make(url.Values)

			for queryParam, values := range tc.givenQueryTokens {
				for _, v := range values {
					q.Add(queryParam, v)
				}
			}

			f := make(url.Values)

			for formKey, values := range tc.givenFormTokens {
				for _, v := range values {
					f.Add(formKey, v)
				}
			}

			var req *http.Request

			switch tc.givenMethod {
			case http.MethodGet:
				req = httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
			case http.MethodPost, http.MethodPut:
				req = httptest.NewRequest(http.MethodPost, "/?"+q.Encode(), strings.NewReader(f.Encode()))
				req.Header.Add(echox.HeaderContentType, echox.MIMEApplicationForm)
			}

			for header, values := range tc.givenHeaderTokens {
				for _, v := range values {
					req.Header.Add(header, v)
				}
			}

			if tc.givenCSRFCookie != "" {
				req.Header.Set(echox.HeaderCookie, "_csrf="+tc.givenCSRFCookie)
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			config := CSRFConfig{
				TokenLookup: tc.whenTokenLookup,
				CookieName:  tc.whenCookieName,
			}

			csrf, err := config.ToMiddleware()
			if tc.expectToMiddlewareError != "" {
				assert.EqualError(t, err, tc.expectToMiddlewareError)
				return
			} else if err != nil {
				assert.NoError(t, err)
			}

			h := csrf(func(c echox.Context) error {
				return c.String(http.StatusOK, "test")
			})

			err = h(c)
			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCSRF(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	csrf := CSRF()
	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// Generate CSRF token
	h(c)
	assert.Contains(t, rec.Header().Get(echox.HeaderSetCookie), "_csrf")
}

func TestMustCSRFWithConfig(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	csrf := CSRFWithConfig(CSRFConfig{
		TokenLength: 16,
	})
	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	// Generate CSRF token
	h(c)
	assert.Contains(t, rec.Header().Get(echox.HeaderSetCookie), "_csrf")

	// Without CSRF cookie
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	assert.Error(t, h(c))

	// Empty/invalid CSRF token
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	req.Header.Set(echox.HeaderXCSRFToken, "")
	assert.Error(t, h(c))

	// Valid CSRF token
	token := randomString(16)
	req.Header.Set(echox.HeaderCookie, "_csrf="+token)
	req.Header.Set(echox.HeaderXCSRFToken, token)

	if assert.NoError(t, h(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestCSRFSetSameSiteMode(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	csrf := CSRFWithConfig(CSRFConfig{
		CookieSameSite: http.SameSiteStrictMode,
	})

	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	r := h(c)
	assert.NoError(t, r)
	assert.Regexp(t, "SameSite=Strict", rec.Header()["Set-Cookie"])
}

func TestCSRFWithoutSameSiteMode(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	csrf := CSRFWithConfig(CSRFConfig{})

	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	r := h(c)
	assert.NoError(t, r)
	assert.NotRegexp(t, "SameSite=", rec.Header()["Set-Cookie"])
}

func TestCSRFWithSameSiteDefaultMode(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	csrf := CSRFWithConfig(CSRFConfig{
		CookieSameSite: http.SameSiteDefaultMode,
	})

	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	r := h(c)
	assert.NoError(t, r)
	assert.NotRegexp(t, "SameSite=", rec.Header()["Set-Cookie"])
}

func TestCSRFWithSameSiteModeNone(t *testing.T) {
	e := echox.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	csrf, err := CSRFConfig{
		CookieSameSite: http.SameSiteNoneMode,
	}.ToMiddleware()
	assert.NoError(t, err)

	h := csrf(func(c echox.Context) error {
		return c.String(http.StatusOK, "test")
	})

	r := h(c)
	assert.NoError(t, r)
	assert.Regexp(t, "SameSite=None", rec.Header()["Set-Cookie"])
	assert.Regexp(t, "Secure", rec.Header()["Set-Cookie"])
}

func TestCSRFConfig_skipper(t *testing.T) {
	var testCases = []struct {
		name          string
		whenSkip      bool
		expectCookies int
	}{
		{
			name:          "do skip",
			whenSkip:      true,
			expectCookies: 0,
		},
		{
			name:          "do not skip",
			whenSkip:      false,
			expectCookies: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			csrf := CSRFWithConfig(CSRFConfig{
				Skipper: func(c echox.Context) bool {
					return tc.whenSkip
				},
			})

			h := csrf(func(c echox.Context) error {
				return c.String(http.StatusOK, "test")
			})

			r := h(c)
			assert.NoError(t, r)

			cookie := rec.Header()["Set-Cookie"]
			assert.Len(t, cookie, tc.expectCookies)
		})
	}
}

func TestCSRFErrorHandling(t *testing.T) {
	cfg := CSRFConfig{
		ErrorHandler: func(c echox.Context, err error) error {
			return echox.NewHTTPError(http.StatusTeapot, "error_handler_executed")
		},
	}

	e := echox.New()
	e.POST("/", func(c echox.Context) error {
		return c.String(http.StatusNotImplemented, "should not end up here")
	})

	e.Use(CSRFWithConfig(cfg))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	assert.Equal(t, http.StatusTeapot, res.Code)
	assert.Equal(t, "{\"message\":\"error_handler_executed\"}\n", res.Body.String())
}
