package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/theopenlane/echox"
)

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper Skipper

	// AllowOrigins determines the value of the Access-Control-Allow-Origin
	// response header.  This header defines a list of origins that may access the
	// resource.  The wildcard characters '*' and '?' are supported and are
	// converted to regex fragments '.*' and '.' accordingly.
	//
	// Security: use extreme caution when handling the origin, and carefully
	// validate any logic. Remember that attackers may register hostile domain names.
	// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
	//
	// Optional. Default value []string{"*"}.
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
	AllowOrigins []string `yaml:"allow_origins"`

	// AllowOriginFunc is a custom function to validate the origin. It takes the
	// origin as an argument and returns true if allowed or false otherwise. If
	// an error is returned, it is returned by the handler. If this option is
	// set, AllowOrigins is ignored.
	//
	// Security: use extreme caution when handling the origin, and carefully
	// validate any logic. Remember that attackers may register hostile domain names.
	// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
	//
	// Optional.
	AllowOriginFunc func(origin string) (bool, error) `yaml:"-"`

	// AllowMethods determines the value of the Access-Control-Allow-Methods
	// response header.  This header specified the list of methods allowed when
	// accessing the resource.  This is used in response to a preflight request.
	//
	// Optional. Default value DefaultCORSConfig.AllowMethods.
	// If `allowMethods` is left empty, this middleware will fill for preflight
	// request `Access-Control-Allow-Methods` header value
	// from `Allow` header that echox.Router set into context.
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods
	AllowMethods []string `yaml:"allow_methods"`

	// AllowHeaders determines the value of the Access-Control-Allow-Headers
	// response header.  This header is used in response to a preflight request to
	// indicate which HTTP headers can be used when making the actual request.
	//
	// Optional. Default value []string{}.
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
	AllowHeaders []string `yaml:"allow_headers"`

	// AllowCredentials determines the value of the
	// Access-Control-Allow-Credentials response header.  This header indicates
	// whether or not the response to the request can be exposed when the
	// credentials mode (Request.credentials) is true. When used as part of a
	// response to a preflight request, this indicates whether or not the actual
	// request can be made using credentials.  See also
	// [MDN: Access-Control-Allow-Credentials].
	//
	// Optional. Default value false, in which case the header is not set.
	//
	// Security: avoid using `AllowCredentials = true` with `AllowOrigins = *`.
	// See "Exploiting CORS misconfigurations for Bitcoins and bounties",
	// https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
	AllowCredentials bool `yaml:"allow_credentials"`

	// UnsafeWildcardOriginWithAllowCredentials UNSAFE/INSECURE: allows wildcard '*' origin to be used with AllowCredentials
	// flag. In that case we consider any origin allowed and send it back to the client with `Access-Control-Allow-Origin` header.
	//
	// This is INSECURE and potentially leads to [cross-origin](https://portswigger.net/research/exploiting-cors-misconfigurations-for-bitcoins-and-bounties)
	// attacks. See: https://github.com/labstack/echo/issues/2400 for discussion on the subject.
	//
	// Optional. Default value is false.
	UnsafeWildcardOriginWithAllowCredentials bool `yaml:"unsafe_wildcard_origin_with_allow_credentials"`

	// ExposeHeaders determines the value of Access-Control-Expose-Headers, which
	// defines a list of headers that clients are allowed to access.
	//
	// Optional. Default value []string{}, in which case the header is not set.
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Expose-Header
	ExposeHeaders []string `yaml:"expose_headers"`

	// MaxAge determines the value of the Access-Control-Max-Age response header.
	// This header indicates how long (in seconds) the results of a preflight
	// request can be cached.
	// The header is set only if MaxAge != 0, negative value sends "0" which instructs browsers not to cache that response.
	//
	// Optional. Default value 0 - meaning header is not sent.
	//
	// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age
	MaxAge int `yaml:"max_age"`
}

// DefaultCORSConfig is the default CORS middleware config.
var DefaultCORSConfig = CORSConfig{
	Skipper:      DefaultSkipper,
	AllowOrigins: []string{"*"},
	AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
}

// CORS returns a Cross-Origin Resource Sharing (CORS) middleware.
// See also [MDN: Cross-Origin Resource Sharing (CORS)].
//
// Security: Poorly configured CORS can compromise security because it allows
// relaxation of the browser's Same-Origin policy.  See [Exploiting CORS
// misconfigurations for Bitcoins and bounties] and [Portswigger: Cross-origin
// resource sharing (CORS)] for more details.
//
// [MDN: Cross-Origin Resource Sharing (CORS)]: https://developer.mozilla.org/en/docs/Web/HTTP/Access_control_CORS
// [Exploiting CORS misconfigurations for Bitcoins and bounties]: https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
// [Portswigger: Cross-origin resource sharing (CORS)]: https://portswigger.net/web-security/cors
func CORS() echox.MiddlewareFunc {
	return CORSWithConfig(DefaultCORSConfig)
}

// CORSWithConfig returns a CORS middleware with config or panics on invalid configuration.
// See: [CORS].
func CORSWithConfig(config CORSConfig) echox.MiddlewareFunc {
	return toMiddlewareOrPanic(config)
}

// ToMiddleware converts CORSConfig to middleware or returns an error for invalid configuration
func (config CORSConfig) ToMiddleware() (echox.MiddlewareFunc, error) {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultCORSConfig.Skipper
	}

	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}

	hasCustomAllowMethods := true
	if len(config.AllowMethods) == 0 {
		hasCustomAllowMethods = false
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	allowOriginPatterns := make([]*regexp.Regexp, 0, len(config.AllowOrigins))

	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			continue // "*" is handled differently and does not need regexp
		}

		pattern := regexp.QuoteMeta(origin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = strings.ReplaceAll(pattern, "\\?", ".")
		pattern = "^" + pattern + "$"

		re, err := regexp.Compile(pattern)
		if err != nil {
			panic(err)
		}

		allowOriginPatterns = append(allowOriginPatterns, re)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")

	maxAge := "0"
	if config.MaxAge > 0 {
		maxAge = strconv.Itoa(config.MaxAge)
	}

	return func(next echox.HandlerFunc) echox.HandlerFunc {
		return func(c echox.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()
			origin := req.Header.Get(echox.HeaderOrigin)
			allowOrigin := ""

			res.Header().Add(echox.HeaderVary, echox.HeaderOrigin)

			// Preflight request is an OPTIONS request, using three HTTP request headers: Access-Control-Request-Method,
			// Access-Control-Request-Headers, and the Origin header. See: https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
			// For simplicity we just consider method type and later `Origin` header.
			preflight := req.Method == http.MethodOptions

			// Although router adds special handler in case of OPTIONS method we avoid calling next for OPTIONS in this middleware
			// as CORS requests do not have cookies / authentication headers by default, so we could get stuck in auth
			// middlewares by calling next(c).
			// But we still want to send `Allow` header as response in case of Non-CORS OPTIONS request as router default
			// handler does.
			routerAllowMethods := ""

			if preflight {
				tmpAllowMethods, ok := c.Get(echox.ContextKeyHeaderAllow).(string)
				if ok && tmpAllowMethods != "" {
					routerAllowMethods = tmpAllowMethods
					c.Response().Header().Set(echox.HeaderAllow, routerAllowMethods)
				}
			}

			// No Origin provided. This is (probably) not request from actual browser - proceed executing middleware chain
			if origin == "" {
				if !preflight {
					return next(c)
				}

				return c.NoContent(http.StatusNoContent)
			}

			if config.AllowOriginFunc != nil {
				allowed, err := config.AllowOriginFunc(origin)
				if err != nil {
					return err
				}

				if allowed {
					allowOrigin = origin
				}
			} else {
				// Check allowed origins
				for _, o := range config.AllowOrigins {
					if o == "*" && config.AllowCredentials && config.UnsafeWildcardOriginWithAllowCredentials {
						allowOrigin = origin
						break
					}

					if o == "*" || o == origin {
						allowOrigin = o
						break
					}

					if matchSubdomain(origin, o) {
						allowOrigin = origin
						break
					}
				}

				checkPatterns := false

				if allowOrigin == "" {
					// to avoid regex cost by invalid (long) domains (253 is domain name max limit)
					if len(origin) <= (253+3+5) && strings.Contains(origin, "://") {
						checkPatterns = true
					}
				}

				if checkPatterns {
					for _, re := range allowOriginPatterns {
						if match := re.MatchString(origin); match {
							allowOrigin = origin
							break
						}
					}
				}
			}

			// Origin not allowed
			if allowOrigin == "" {
				if !preflight {
					// https://github.com/labstack/echo/pull/2732/files
					return echox.ErrUnauthorized
				}

				return c.NoContent(http.StatusNoContent)
			}

			res.Header().Set(echox.HeaderAccessControlAllowOrigin, allowOrigin)

			if config.AllowCredentials {
				res.Header().Set(echox.HeaderAccessControlAllowCredentials, "true")
			}

			// Simple request
			if !preflight {
				if exposeHeaders != "" {
					res.Header().Set(echox.HeaderAccessControlExposeHeaders, exposeHeaders)
				}

				return next(c)
			}

			// Preflight request
			res.Header().Add(echox.HeaderVary, echox.HeaderAccessControlRequestMethod)
			res.Header().Add(echox.HeaderVary, echox.HeaderAccessControlRequestHeaders)

			if !hasCustomAllowMethods && routerAllowMethods != "" {
				res.Header().Set(echox.HeaderAccessControlAllowMethods, routerAllowMethods)
			} else {
				res.Header().Set(echox.HeaderAccessControlAllowMethods, allowMethods)
			}

			if allowHeaders != "" {
				res.Header().Set(echox.HeaderAccessControlAllowHeaders, allowHeaders)
			} else {
				h := req.Header.Get(echox.HeaderAccessControlRequestHeaders)
				if h != "" {
					res.Header().Set(echox.HeaderAccessControlAllowHeaders, h)
				}
			}

			if config.MaxAge != 0 {
				res.Header().Set(echox.HeaderAccessControlMaxAge, maxAge)
			}

			return c.NoContent(http.StatusNoContent)
		}
	}, nil
}
