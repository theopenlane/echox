package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/theopenlane/echox"
)

func TestCORS(t *testing.T) {
	var testCases = []struct {
		name             string
		givenMW          echox.MiddlewareFunc
		whenMethod       string
		whenHeaders      map[string]string
		expectHeaders    map[string]string
		notExpectHeaders map[string]string
	}{
		{
			name:          "ok, wildcard origin",
			whenHeaders:   map[string]string{echox.HeaderOrigin: "localhost"},
			expectHeaders: map[string]string{echox.HeaderAccessControlAllowOrigin: "*"},
		},
		{
			name:             "ok, wildcard AllowedOrigin with no Origin header in request",
			notExpectHeaders: map[string]string{echox.HeaderAccessControlAllowOrigin: ""},
		},
		{
			name: "ok, specific AllowOrigins and AllowCredentials",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins:     []string{"localhost"},
				AllowCredentials: true,
				MaxAge:           3600,
			}),
			whenHeaders: map[string]string{echox.HeaderOrigin: "localhost"},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:      "localhost",
				echox.HeaderAccessControlAllowCredentials: "true",
			},
		},
		{
			name: "ok, preflight request with matching origin for `AllowOrigins`",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins:     []string{"localhost"},
				AllowCredentials: true,
				MaxAge:           3600,
			}),
			whenMethod: http.MethodOptions,
			whenHeaders: map[string]string{
				echox.HeaderOrigin:      "localhost",
				echox.HeaderContentType: echox.MIMEApplicationJSON,
			},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:      "localhost",
				echox.HeaderAccessControlAllowMethods:     "GET,HEAD,PUT,PATCH,POST,DELETE",
				echox.HeaderAccessControlAllowCredentials: "true",
				echox.HeaderAccessControlMaxAge:           "3600",
			},
		},
		{
			name: "ok, preflight request with wildcard `AllowOrigins` and `AllowCredentials` true",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				MaxAge:           3600,
			}),
			whenMethod: http.MethodOptions,
			whenHeaders: map[string]string{
				echox.HeaderOrigin:      "localhost",
				echox.HeaderContentType: echox.MIMEApplicationJSON,
			},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:      "*", // Note: browsers will ignore and complain about responses having `*`
				echox.HeaderAccessControlAllowMethods:     "GET,HEAD,PUT,PATCH,POST,DELETE",
				echox.HeaderAccessControlAllowCredentials: "true",
				echox.HeaderAccessControlMaxAge:           "3600",
			},
		},
		{
			name: "ok, preflight request with wildcard `AllowOrigins` and `AllowCredentials` false",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: false, // important for this testcase
				MaxAge:           3600,
			}),
			whenMethod: http.MethodOptions,
			whenHeaders: map[string]string{
				echox.HeaderOrigin:      "localhost",
				echox.HeaderContentType: echox.MIMEApplicationJSON,
			},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:  "*",
				echox.HeaderAccessControlAllowMethods: "GET,HEAD,PUT,PATCH,POST,DELETE",
				echox.HeaderAccessControlMaxAge:       "3600",
			},
			notExpectHeaders: map[string]string{
				echox.HeaderAccessControlAllowCredentials: "",
			},
		},
		{
			name: "ok, INSECURE preflight request with wildcard `AllowOrigins` and `AllowCredentials` true",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins:                             []string{"*"},
				AllowCredentials:                         true,
				UnsafeWildcardOriginWithAllowCredentials: true, // important for this testcase
				MaxAge:                                   3600,
			}),
			whenMethod: http.MethodOptions,
			whenHeaders: map[string]string{
				echox.HeaderOrigin:      "localhost",
				echox.HeaderContentType: echox.MIMEApplicationJSON,
			},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:      "localhost", // This could end up as cross-origin attack
				echox.HeaderAccessControlAllowMethods:     "GET,HEAD,PUT,PATCH,POST,DELETE",
				echox.HeaderAccessControlAllowCredentials: "true",
				echox.HeaderAccessControlMaxAge:           "3600",
			},
		},
		{
			name: "ok, preflight request with Access-Control-Request-Headers",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins: []string{"*"},
			}),
			whenMethod: http.MethodOptions,
			whenHeaders: map[string]string{
				echox.HeaderOrigin:                      "localhost",
				echox.HeaderContentType:                 echox.MIMEApplicationJSON,
				echox.HeaderAccessControlRequestHeaders: "Special-Request-Header",
			},
			expectHeaders: map[string]string{
				echox.HeaderAccessControlAllowOrigin:  "*",
				echox.HeaderAccessControlAllowHeaders: "Special-Request-Header",
				echox.HeaderAccessControlAllowMethods: "GET,HEAD,PUT,PATCH,POST,DELETE",
			},
		},
		{
			name: "ok, preflight request with `AllowOrigins` which allow all subdomains aaa with *",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins: []string{"http://*.example.com"},
			}),
			whenMethod:    http.MethodOptions,
			whenHeaders:   map[string]string{echox.HeaderOrigin: "http://aaa.example.com"},
			expectHeaders: map[string]string{echox.HeaderAccessControlAllowOrigin: "http://aaa.example.com"},
		},
		{
			name: "ok, preflight request with `AllowOrigins` which allow all subdomains bbb with *",
			givenMW: CORSWithConfig(CORSConfig{
				AllowOrigins: []string{"http://*.example.com"},
			}),
			whenMethod:    http.MethodOptions,
			whenHeaders:   map[string]string{echox.HeaderOrigin: "http://bbb.example.com"},
			expectHeaders: map[string]string{echox.HeaderAccessControlAllowOrigin: "http://bbb.example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()

			mw := CORS()
			if tc.givenMW != nil {
				mw = tc.givenMW
			}

			h := mw(func(c echox.Context) error {
				return nil
			})

			method := http.MethodGet
			if tc.whenMethod != "" {
				method = tc.whenMethod
			}

			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			for k, v := range tc.whenHeaders {
				req.Header.Set(k, v)
			}

			err := h(c)

			assert.NoError(t, err)

			header := rec.Header()
			for k, v := range tc.expectHeaders {
				assert.Equal(t, v, header.Get(k), "header: `%v` should be `%v`", k, v)
			}

			for k, v := range tc.notExpectHeaders {
				if v == "" {
					assert.Len(t, header.Values(k), 0, "header: `%v` should not be set", k)
				} else {
					assert.NotEqual(t, v, header.Get(k), "header: `%v` should not be `%v`", k, v)
				}
			}
		})
	}
}

func Test_allowOriginScheme(t *testing.T) {
	tests := []struct {
		domain, pattern string
		expected        bool
	}{
		{
			domain:   "http://example.com",
			pattern:  "http://example.com",
			expected: true,
		},
		{
			domain:   "https://example.com",
			pattern:  "https://example.com",
			expected: true,
		},
		{
			domain:   "http://example.com",
			pattern:  "https://example.com",
			expected: false,
		},
		{
			domain:   "https://example.com",
			pattern:  "http://example.com",
			expected: false,
		},
	}

	e := echox.New()

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		req.Header.Set(echox.HeaderOrigin, tt.domain)
		cors := CORSWithConfig(CORSConfig{
			AllowOrigins: []string{tt.pattern},
		})
		h := cors(func(c echox.Context) error { return echox.ErrNotFound })
		h(c)

		if tt.expected {
			assert.Equal(t, tt.domain, rec.Header().Get(echox.HeaderAccessControlAllowOrigin))
		} else {
			assert.NotContains(t, rec.Header(), echox.HeaderAccessControlAllowOrigin)
		}
	}
}

func Test_allowOriginSubdomain(t *testing.T) {
	tests := []struct {
		domain, pattern string
		expected        bool
	}{
		{
			domain:   "http://aaa.example.com",
			pattern:  "http://*.example.com",
			expected: true,
		},
		{
			domain:   "http://bbb.aaa.example.com",
			pattern:  "http://*.example.com",
			expected: true,
		},
		{
			domain:   "http://bbb.aaa.example.com",
			pattern:  "http://*.aaa.example.com",
			expected: true,
		},
		{
			domain:   "http://aaa.example.com:8080",
			pattern:  "http://*.example.com:8080",
			expected: true,
		},

		{
			domain:   "http://fuga.hoge.com",
			pattern:  "http://*.example.com",
			expected: false,
		},
		{
			domain:   "http://ccc.bbb.example.com",
			pattern:  "http://*.aaa.example.com",
			expected: false,
		},
		{
			domain: `http://1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890\
		  .1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890\
		  .1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890\
		  .1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.example.com`,
			pattern:  "http://*.example.com",
			expected: false,
		},
		{
			domain:   `http://1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.1234567890.example.com`,
			pattern:  "http://*.example.com",
			expected: false,
		},
		{
			domain:   "http://ccc.bbb.example.com",
			pattern:  "http://example.com",
			expected: false,
		},
		{
			domain:   "https://prod-preview--aaa.bbb.com",
			pattern:  "https://*--aaa.bbb.com",
			expected: true,
		},
		{
			domain:   "http://ccc.bbb.example.com",
			pattern:  "http://*.example.com",
			expected: true,
		},
		{
			domain:   "http://ccc.bbb.example.com",
			pattern:  "http://foo.[a-z]*.example.com",
			expected: false,
		},
	}

	e := echox.New()

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		req.Header.Set(echox.HeaderOrigin, tt.domain)
		cors := CORSWithConfig(CORSConfig{
			AllowOrigins: []string{tt.pattern},
		})
		h := cors(func(c echox.Context) error { return echox.ErrNotFound })
		h(c)

		if tt.expected {
			assert.Equal(t, tt.domain, rec.Header().Get(echox.HeaderAccessControlAllowOrigin))
		} else {
			assert.NotContains(t, rec.Header(), echox.HeaderAccessControlAllowOrigin)
		}
	}
}

func TestCORSWithConfig_AllowMethods(t *testing.T) {
	var testCases = []struct {
		name            string
		allowOrigins    []string
		allowContextKey string

		whenOrigin       string
		whenAllowMethods []string

		expectAllow                     string
		expectAccessControlAllowMethods string
	}{
		{
			name:             "custom AllowMethods, preflight, no origin, sets only allow header from context key",
			allowContextKey:  "OPTIONS, GET",
			whenAllowMethods: []string{http.MethodGet, http.MethodHead},
			whenOrigin:       "",
			expectAllow:      "OPTIONS, GET",
		},
		{
			name:             "default AllowMethods, preflight, no origin, no allow header in context key and in response",
			allowContextKey:  "",
			whenAllowMethods: nil,
			whenOrigin:       "",
			expectAllow:      "",
		},
		{
			name:                            "custom AllowMethods, preflight, existing origin, sets both headers different values",
			allowContextKey:                 "OPTIONS, GET",
			whenAllowMethods:                []string{http.MethodGet, http.MethodHead},
			whenOrigin:                      "http://google.com",
			expectAllow:                     "OPTIONS, GET",
			expectAccessControlAllowMethods: "GET,HEAD",
		},
		{
			name:                            "default AllowMethods, preflight, existing origin, sets both headers",
			allowContextKey:                 "OPTIONS, GET",
			whenAllowMethods:                nil,
			whenOrigin:                      "http://google.com",
			expectAllow:                     "OPTIONS, GET",
			expectAccessControlAllowMethods: "OPTIONS, GET",
		},
		{
			name:                            "default AllowMethods, preflight, existing origin, no allows, sets only CORS allow methods",
			allowContextKey:                 "",
			whenAllowMethods:                nil,
			whenOrigin:                      "http://google.com",
			expectAllow:                     "",
			expectAccessControlAllowMethods: "GET,HEAD,PUT,PATCH,POST,DELETE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()
			e.GET("/test", func(c echox.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			cors := CORSWithConfig(CORSConfig{
				AllowOrigins: tc.allowOrigins,
				AllowMethods: tc.whenAllowMethods,
			})

			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			req.Header.Set(echox.HeaderOrigin, tc.whenOrigin)

			if tc.allowContextKey != "" {
				c.Set(echox.ContextKeyHeaderAllow, tc.allowContextKey)
			}

			h := cors(func(c echox.Context) error {
				return c.String(http.StatusOK, "OK")
			})
			h(c)

			assert.Equal(t, tc.expectAllow, rec.Header().Get(echox.HeaderAllow))
			assert.Equal(t, tc.expectAccessControlAllowMethods, rec.Header().Get(echox.HeaderAccessControlAllowMethods))
		})
	}
}

func TestCorsHeaders(t *testing.T) {
	tests := []struct {
		name              string
		originDomain      string
		method            string
		allowedOrigin     string
		expected          bool
		expectStatus      int
		expectAllowHeader string
	}{
		{
			name:          "non-preflight request, allow any origin, missing origin header = no CORS logic done",
			originDomain:  "",
			allowedOrigin: "*",
			method:        http.MethodGet,
			expected:      false,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "non-preflight request, allow any origin, specific origin domain",
			originDomain:  "http://example.com",
			allowedOrigin: "*",
			method:        http.MethodGet,
			expected:      true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "non-preflight request, allow specific origin, missing origin header = no CORS logic done",
			originDomain:  "", // Request does not have Origin header
			allowedOrigin: "http://example.com",
			method:        http.MethodGet,
			expected:      false,
			expectStatus:  http.StatusOK,
		},
		//{
		//	name:          "non-preflight request, allow specific origin, different origin header = CORS logic failure",
		//	originDomain:  "http://bar.com",
		//	allowedOrigin: "http://example.com",
		//	method:        http.MethodGet,
		//	expected:      false,
		//	expectStatus:  http.StatusOK,
		//},
		{
			name:          "non-preflight request, allow specific origin, matching origin header = CORS logic done",
			originDomain:  "http://example.com",
			allowedOrigin: "http://example.com",
			method:        http.MethodGet,
			expected:      true,
			expectStatus:  http.StatusOK,
		},
		{
			name:              "preflight, allow any origin, missing origin header = no CORS logic done",
			originDomain:      "", // Request does not have Origin header
			allowedOrigin:     "*",
			method:            http.MethodOptions,
			expected:          false,
			expectStatus:      http.StatusNoContent,
			expectAllowHeader: "OPTIONS, GET, POST",
		},
		{
			name:              "preflight, allow any origin, existing origin header = CORS logic done",
			originDomain:      "http://example.com",
			allowedOrigin:     "*",
			method:            http.MethodOptions,
			expected:          true,
			expectStatus:      http.StatusNoContent,
			expectAllowHeader: "OPTIONS, GET, POST",
		},
		{
			name:              "preflight, allow any origin, missing origin header = no CORS logic done",
			originDomain:      "", // Request does not have Origin header
			allowedOrigin:     "http://example.com",
			method:            http.MethodOptions,
			expected:          false,
			expectStatus:      http.StatusNoContent,
			expectAllowHeader: "OPTIONS, GET, POST",
		},
		{
			name:              "preflight, allow specific origin, different origin header = no CORS logic done",
			originDomain:      "http://bar.com",
			allowedOrigin:     "http://example.com",
			method:            http.MethodOptions,
			expected:          false,
			expectStatus:      http.StatusNoContent,
			expectAllowHeader: "OPTIONS, GET, POST",
		},
		{
			name:              "preflight, allow specific origin, matching origin header = CORS logic done",
			originDomain:      "http://example.com",
			allowedOrigin:     "http://example.com",
			method:            http.MethodOptions,
			expected:          true,
			expectStatus:      http.StatusNoContent,
			expectAllowHeader: "OPTIONS, GET, POST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echox.New()

			e.Use(CORSWithConfig(CORSConfig{
				AllowOrigins: []string{tc.allowedOrigin},
				//AllowCredentials: true,
				//MaxAge:           3600,
			}))

			e.GET("/", func(c echox.Context) error {
				return c.String(http.StatusOK, "OK")
			})
			e.POST("/", func(c echox.Context) error {
				return c.String(http.StatusCreated, "OK")
			})

			req := httptest.NewRequest(tc.method, "/", nil)
			rec := httptest.NewRecorder()

			if tc.originDomain != "" {
				req.Header.Set(echox.HeaderOrigin, tc.originDomain)
			}

			// we run through whole Echo handler chain to see how CORS works with Router OPTIONS handler
			e.ServeHTTP(rec, req)

			assert.Equal(t, echox.HeaderOrigin, rec.Header().Get(echox.HeaderVary))
			assert.Equal(t, tc.expectAllowHeader, rec.Header().Get(echox.HeaderAllow))
			assert.Equal(t, tc.expectStatus, rec.Code)

			expectedAllowOrigin := ""
			if tc.allowedOrigin == "*" {
				expectedAllowOrigin = "*"
			} else {
				expectedAllowOrigin = tc.originDomain
			}

			switch {
			case tc.expected && tc.method == http.MethodOptions:
				assert.Contains(t, rec.Header(), echox.HeaderAccessControlAllowMethods)
				assert.Equal(t, expectedAllowOrigin, rec.Header().Get(echox.HeaderAccessControlAllowOrigin))

				assert.Equal(t, 3, len(rec.Header()[echox.HeaderVary]))

			case tc.expected && tc.method == http.MethodGet:
				assert.Equal(t, expectedAllowOrigin, rec.Header().Get(echox.HeaderAccessControlAllowOrigin))
				assert.Equal(t, 1, len(rec.Header()[echox.HeaderVary])) // Vary: Origin
			default:
				assert.NotContains(t, rec.Header(), echox.HeaderAccessControlAllowOrigin)
				assert.Equal(t, 1, len(rec.Header()[echox.HeaderVary])) // Vary: Origin
			}
		})
	}
}

func Test_allowOriginFunc(t *testing.T) {
	returnTrue := func(origin string) (bool, error) {
		return true, nil
	}
	returnFalse := func(origin string) (bool, error) {
		return false, nil
	}
	returnError := func(origin string) (bool, error) {
		return true, errors.New("this is a test error")
	}

	allowOriginFuncs := []func(origin string) (bool, error){
		returnTrue,
		returnFalse,
		returnError,
	}

	const origin = "http://example.com"

	e := echox.New()

	for _, allowOriginFunc := range allowOriginFuncs {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		req.Header.Set(echox.HeaderOrigin, origin)

		cors, err := CORSConfig{AllowOriginFunc: allowOriginFunc}.ToMiddleware()
		assert.NoError(t, err)

		h := cors(func(c echox.Context) error { return echox.ErrNotFound })
		err = h(c)

		expected, expectedErr := allowOriginFunc(origin)
		if expectedErr != nil {
			assert.Equal(t, expectedErr, err)
			assert.Equal(t, "", rec.Header().Get(echox.HeaderAccessControlAllowOrigin))

			continue
		}

		if expected {
			assert.Equal(t, origin, rec.Header().Get(echox.HeaderAccessControlAllowOrigin))
		} else {
			assert.Equal(t, "", rec.Header().Get(echox.HeaderAccessControlAllowOrigin))
		}
	}
}
