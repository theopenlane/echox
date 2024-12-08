package echocontext

import (
	"context"

	"github.com/theopenlane/utils/contextx"

	echo "github.com/theopenlane/echox"
)

// CustomContext contains the echo.Context and request context.Context
type CustomContext struct {
	echo.Context
	ctx context.Context
}

// EchoContextToContextMiddleware is the middleware that adds the echo.Context to the parent context
func EchoContextToContextMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := contextx.With(c.Request().Context(), c)

			c.SetRequest(c.Request().WithContext(ctx))

			cc := &CustomContext{c, ctx}

			return next(cc)
		}
	}
}

// EchoContextFromContext gets the echo.Context from the parent context
func EchoContextFromContext(ctx context.Context) (echo.Context, error) {
	ec, ok := contextx.From[echo.Context](ctx)
	if !ok {
		return nil, ErrUnableToRetrieveEchoContext
	}

	return ec, nil
}
