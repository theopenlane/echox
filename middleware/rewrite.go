package middleware

import (
	"errors"
	"regexp"

	"github.com/theopenlane/echox"
)

// RewriteConfig defines the config for Rewrite middleware.
type RewriteConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper Skipper

	// Rules defines the URL path rewrite rules. The values captured in asterisk can be
	// retrieved by index e.g. $1, $2 and so on.
	// Example:
	// "/old":              "/new",
	// "/api/*":            "/$1",
	// "/js/*":             "/public/javascripts/$1",
	// "/users/*/orders/*": "/user/$1/order/$2",
	// Required.
	Rules map[string]string

	// RegexRules defines the URL path rewrite rules using regexp.Rexexp with captures
	// Every capture group in the values can be retrieved by index e.g. $1, $2 and so on.
	// Example:
	// "^/old/[0.9]+/":     "/new",
	// "^/api/.+?/(.*)":     "/v2/$1",
	RegexRules map[*regexp.Regexp]string
}

// Rewrite returns a Rewrite middleware.
//
// Rewrite middleware rewrites the URL path based on the provided rules.
func Rewrite(rules map[string]string) echox.MiddlewareFunc {
	c := RewriteConfig{}
	c.Rules = rules

	return RewriteWithConfig(c)
}

// RewriteWithConfig returns a Rewrite middleware or panics on invalid configuration.
//
// Rewrite middleware rewrites the URL path based on the provided rules.
func RewriteWithConfig(config RewriteConfig) echox.MiddlewareFunc {
	return toMiddlewareOrPanic(config)
}

// ToMiddleware converts RewriteConfig to middleware or returns an error for invalid configuration
func (config RewriteConfig) ToMiddleware() (echox.MiddlewareFunc, error) {
	if config.Skipper == nil {
		config.Skipper = DefaultSkipper
	}

	if config.Rules == nil && config.RegexRules == nil {
		return nil, errors.New("echo rewrite middleware requires url path rewrite rules or regex rules")
	}

	if config.RegexRules == nil {
		config.RegexRules = make(map[*regexp.Regexp]string)
	}

	for k, v := range rewriteRulesRegex(config.Rules) {
		config.RegexRules[k] = v
	}

	return func(next echox.HandlerFunc) echox.HandlerFunc {
		return func(c echox.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			if err := rewriteURL(config.RegexRules, c.Request()); err != nil {
				return err
			}

			return next(c)
		}
	}, nil
}
