package echocontext_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	echo "github.com/theopenlane/echox"
	"github.com/theopenlane/echox/middleware/echocontext"
	"github.com/theopenlane/utils/contextx"
)

// MockEchoContext is a mock implementation of echo.Context
type MockEchoContext struct {
	echo.Context
}

func TestEchoContextToContextMiddleware(t *testing.T) {
	// Create a new Echo instance
	e := echo.New()

	// Create a mock echo.Context
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mockEchoContext := e.NewContext(req, rec)

	// Create a middleware
	middleware := echocontext.EchoContextToContextMiddleware()

	// Create a handler to test the middleware
	handler := middleware(func(c echo.Context) error {
		// Retrieve the context from the request
		ctx := c.Request().Context()
		if ctx == nil {
			t.Fatal("Request context is nil")
		}

		// Retrieve the echo.Context from the context
		retrievedEchoContext, err := echocontext.EchoContextFromContext(ctx)
		require.NoError(t, err)

		// Assert that the retrieved echo.Context is the same as the mock echo.Context
		assert.Equal(t, mockEchoContext, retrievedEchoContext)

		return nil
	})

	// Call the handler with the mock echo.Context
	err := handler(mockEchoContext)
	require.NoError(t, err)
}

func TestEchoContextFromContext(t *testing.T) {
	echoCtx := echocontext.NewTestEchoContext()

	// Create a context with the mock echo.Context
	ctx := contextx.With(context.Background(), echoCtx)

	// Retrieve the echo.Context from the context
	retrievedEchoContext, err := echocontext.EchoContextFromContext(ctx)
	require.NoError(t, err)

	// Assert that the retrieved echo.Context is the same as the mock echo.Context
	assert.Equal(t, echoCtx, retrievedEchoContext)
}

func TestEchoContextFromContext_NotFound(t *testing.T) {
	// Create a context without an echo.Context
	ctx := context.Background()

	// Try to retrieve the echo.Context from the context
	retrievedEchoContext, err := echocontext.EchoContextFromContext(ctx)

	// Assert that an error is returned
	assert.Error(t, err)
	assert.Nil(t, retrievedEchoContext)
}
