package echox

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var myNamedHandler = func(c Context) error {
	return nil
}

type NameStruct struct {
}

func (n *NameStruct) getUsers(c Context) error {
	return nil
}

func TestHandlerName(t *testing.T) {
	myNameFuncVar := func(c Context) error {
		return nil
	}

	tmp := NameStruct{}

	var testCases = []struct {
		name            string
		whenHandlerFunc HandlerFunc
		expect          string
	}{
		{
			name: "ok, func as anonymous func",
			whenHandlerFunc: func(c Context) error {
				return nil
			},
			expect: "github.com/theopenlane/echox.TestHandlerName.func2",
		},
		{
			name:            "ok, func as named function variable",
			whenHandlerFunc: myNameFuncVar,
			expect:          "github.com/theopenlane/echox.TestHandlerName.func1",
		},
		{
			name:            "ok, func as struct method",
			whenHandlerFunc: tmp.getUsers,
			expect:          "github.com/theopenlane/echox.(*NameStruct).getUsers-fm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name := HandlerName(tc.whenHandlerFunc)
			assert.Equal(t, tc.expect, name)
		})
	}
}

func TestHandlerName_differentFuncSameName(t *testing.T) {
	handlerCreator := func(name string) HandlerFunc {
		return func(c Context) error {
			return c.String(http.StatusTeapot, name)
		}
	}
	h1 := handlerCreator("name1")
	assert.Equal(t, "github.com/theopenlane/echox.TestHandlerName_differentFuncSameName.TestHandlerName_differentFuncSameName.func1.func2", HandlerName(h1))

	h2 := handlerCreator("name2")
	assert.Equal(t, "github.com/theopenlane/echox.TestHandlerName_differentFuncSameName.TestHandlerName_differentFuncSameName.func1.func3", HandlerName(h2))
}

func TestRoute_ToRouteInfo(t *testing.T) {
	var testCases = []struct {
		name       string
		given      Route
		whenParams []string
		expect     RouteInfo
	}{
		{
			name: "ok, no params, with name",
			given: Route{
				Method: http.MethodGet,
				Path:   "/test",
				Handler: func(c Context) error {
					return c.String(http.StatusTeapot, "OK")
				},
				Middlewares: nil,
				Name:        "test route",
			},
			expect: routeInfo{
				method: http.MethodGet,
				path:   "/test",
				params: nil,
				name:   "test route",
			},
		},
		{
			name: "ok, params",
			given: Route{
				Method: http.MethodGet,
				Path:   "users/:id/:file", // no slash prefix
				Handler: func(c Context) error {
					return c.String(http.StatusTeapot, "OK")
				},
				Middlewares: nil,
				Name:        "",
			},
			whenParams: []string{"id", "file"},
			expect: routeInfo{
				method: http.MethodGet,
				path:   "users/:id/:file",
				params: []string{"id", "file"},
				name:   "GET:users/:id/:file",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ri := tc.given.ToRouteInfo(tc.whenParams)
			assert.Equal(t, tc.expect, ri)
		})
	}
}

func TestRoute_ToRoute(t *testing.T) {
	route := Route{
		Method: http.MethodGet,
		Path:   "/test",
		Handler: func(c Context) error {
			return c.String(http.StatusTeapot, "OK")
		},
		Middlewares: nil,
		Name:        "test route",
	}

	r := route.ToRoute()
	assert.Equal(t, r.Method, http.MethodGet)
	assert.Equal(t, r.Path, "/test")
	assert.NotNil(t, r.Handler)
	assert.Nil(t, r.Middlewares)
	assert.Equal(t, r.Name, "test route")
}

func TestRoute_ForGroup(t *testing.T) {
	route := Route{
		Method: http.MethodGet,
		Path:   "/test",
		Handler: func(c Context) error {
			return c.String(http.StatusTeapot, "OK")
		},
		Middlewares: nil,
		Name:        "test route",
	}

	mw := func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			return next(c)
		}
	}
	gr := route.ForGroup("/users", []MiddlewareFunc{mw})

	r := gr.ToRoute()
	assert.Equal(t, r.Method, http.MethodGet)
	assert.Equal(t, r.Path, "/users/test")
	assert.NotNil(t, r.Handler)
	assert.Len(t, r.Middlewares, 1)
	assert.Equal(t, r.Name, "test route")
}

func exampleRoutes() Routes {
	return Routes{
		routeInfo{
			method: http.MethodGet,
			path:   "/users",
			params: nil,
			name:   "GET:/users",
		},
		routeInfo{
			method: http.MethodGet,
			path:   "/users/:id",
			params: []string{"id"},
			name:   "GET:/users/:id",
		},
		routeInfo{
			method: http.MethodPost,
			path:   "/users/:id",
			params: []string{"id"},
			name:   "POST:/users/:id",
		},
		routeInfo{
			method: http.MethodDelete,
			path:   "/groups",
			params: nil,
			name:   "non_unique_name",
		},
		routeInfo{
			method: http.MethodPost,
			path:   "/groups",
			params: nil,
			name:   "non_unique_name",
		},
	}
}

func TestRoutes_FindByMethodPath(t *testing.T) {
	var testCases = []struct {
		name        string
		given       Routes
		whenMethod  string
		whenPath    string
		expectName  string
		expectError string
	}{
		{
			name:       "ok, found",
			given:      exampleRoutes(),
			whenMethod: http.MethodGet,
			whenPath:   "/users/:id",
			expectName: "GET:/users/:id",
		},
		{
			name:        "nok, not found",
			given:       exampleRoutes(),
			whenMethod:  http.MethodPut,
			whenPath:    "/users/:id",
			expectName:  "",
			expectError: "route not found by method and path",
		},
		{
			name:        "nok, not found from nil",
			given:       nil,
			whenMethod:  http.MethodGet,
			whenPath:    "/users/:id",
			expectName:  "",
			expectError: "route not found by method and path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ri, err := tc.given.FindByMethodPath(tc.whenMethod, tc.whenPath)

			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
				assert.Nil(t, ri)
			} else {
				assert.NoError(t, err)
			}

			if tc.expectName != "" {
				assert.Equal(t, tc.expectName, ri.Name())
			}
		})
	}
}

func TestRoutes_FilterByMethod(t *testing.T) {
	var testCases = []struct {
		name        string
		given       Routes
		whenMethod  string
		expectNames []string
		expectError string
	}{
		{
			name:        "ok, found",
			given:       exampleRoutes(),
			whenMethod:  http.MethodGet,
			expectNames: []string{"GET:/users", "GET:/users/:id"},
		},
		{
			name:        "nok, not found",
			given:       exampleRoutes(),
			whenMethod:  http.MethodPut,
			expectNames: nil,
			expectError: "route not found by method",
		},
		{
			name:        "nok, not found from nil",
			given:       nil,
			whenMethod:  http.MethodGet,
			expectNames: nil,
			expectError: "route not found by method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ris, err := tc.given.FilterByMethod(tc.whenMethod)

			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}

			if len(tc.expectNames) > 0 {
				assert.Len(t, ris, len(tc.expectNames))

				for _, ri := range ris {
					assert.Contains(t, tc.expectNames, ri.Name())
				}
			} else {
				assert.Nil(t, ris)
			}
		})
	}
}

func TestRoutes_FilterByPath(t *testing.T) {
	var testCases = []struct {
		name        string
		given       Routes
		whenPath    string
		expectNames []string
		expectError string
	}{
		{
			name:        "ok, found",
			given:       exampleRoutes(),
			whenPath:    "/users/:id",
			expectNames: []string{"GET:/users/:id", "POST:/users/:id"},
		},
		{
			name:        "nok, not found",
			given:       exampleRoutes(),
			whenPath:    "/",
			expectNames: nil,
			expectError: "route not found by path",
		},
		{
			name:        "nok, not found from nil",
			given:       nil,
			whenPath:    "/users/:id",
			expectNames: nil,
			expectError: "route not found by path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ris, err := tc.given.FilterByPath(tc.whenPath)

			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}

			if len(tc.expectNames) > 0 {
				assert.Len(t, ris, len(tc.expectNames))

				for _, ri := range ris {
					assert.Contains(t, tc.expectNames, ri.Name())
				}
			} else {
				assert.Nil(t, ris)
			}
		})
	}
}

func TestRoutes_FilterByName(t *testing.T) {
	var testCases = []struct {
		name             string
		given            Routes
		whenName         string
		expectMethodPath []string
		expectError      string
	}{
		{
			name:             "ok, found multiple",
			given:            exampleRoutes(),
			whenName:         "non_unique_name",
			expectMethodPath: []string{"DELETE:/groups", "POST:/groups"},
		},
		{
			name:             "ok, found single",
			given:            exampleRoutes(),
			whenName:         "GET:/users/:id",
			expectMethodPath: []string{"GET:/users/:id"},
		},
		{
			name:             "nok, not found",
			given:            exampleRoutes(),
			whenName:         "/",
			expectMethodPath: nil,
			expectError:      "route not found by name",
		},
		{
			name:             "nok, not found from nil",
			given:            nil,
			whenName:         "/users/:id",
			expectMethodPath: nil,
			expectError:      "route not found by name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ris, err := tc.given.FilterByName(tc.whenName)

			if tc.expectError != "" {
				assert.EqualError(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}

			if len(tc.expectMethodPath) > 0 {
				assert.Len(t, ris, len(tc.expectMethodPath))

				for _, ri := range ris {
					assert.Contains(t, tc.expectMethodPath, fmt.Sprintf("%v:%v", ri.Method(), ri.Path()))
				}
			} else {
				assert.Nil(t, ris)
			}
		})
	}
}

func TestRouteInfo_Reverse(t *testing.T) {
	var testCases = []struct {
		name        string
		givenParams []string
		givenPath   string
		whenParams  []interface{}
		expect      string
	}{
		{
			name:      "ok,static with no params",
			givenPath: "/static",
			expect:    "/static",
		},
		{
			name:        "ok,static with non existent param",
			givenParams: []string{"missing param"},
			givenPath:   "/static",
			whenParams:  []interface{}{"missing param"},
			expect:      "/static",
		},
		{
			name:      "ok, wildcard with no params",
			givenPath: "/static/*",
			expect:    "/static/*",
		},
		{
			name:        "ok, wildcard with params",
			givenParams: []string{"foo.txt"},
			givenPath:   "/static/*",
			whenParams:  []interface{}{"foo.txt"},
			expect:      "/static/foo.txt",
		},
		{
			name:      "ok, single param without param",
			givenPath: "/params/:foo",
			expect:    "/params/:foo",
		},
		{
			name:        "ok, single param with param",
			givenParams: []string{"one"},
			givenPath:   "/params/:foo",
			whenParams:  []interface{}{"one"},
			expect:      "/params/one",
		},
		{
			name:      "ok, multi param without params",
			givenPath: "/params/:foo/bar/:qux",
			expect:    "/params/:foo/bar/:qux",
		},
		{
			name:        "ok, multi param with one param",
			givenParams: []string{"one"},
			givenPath:   "/params/:foo/bar/:qux",
			whenParams:  []interface{}{"one"},
			expect:      "/params/one/bar/:qux",
		},
		{
			name:        "ok, multi param with all params",
			givenParams: []string{"one", "two"},
			givenPath:   "/params/:foo/bar/:qux",
			whenParams:  []interface{}{"one", "two"},
			expect:      "/params/one/bar/two",
		},
		{
			name:        "ok, multi param + wildcard with all params",
			givenParams: []string{"one", "two", "three"},
			givenPath:   "/params/:foo/bar/:qux/*",
			whenParams:  []interface{}{"one", "two", "three"},
			expect:      "/params/one/bar/two/three",
		},
		{
			name:        "ok, backslash is not escaped",
			givenParams: []string{"test"},
			givenPath:   "/a\\b/:x",
			whenParams:  []interface{}{"test"},
			expect:      `/a\b/test`,
		},
		{
			name:        "ok, escaped colon verbs",
			givenParams: []string{"PATCH"},
			givenPath:   "/params\\::customVerb",
			whenParams:  []interface{}{"PATCH"},
			expect:      `/params:PATCH`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := routeInfo{
				path:   tc.givenPath,
				params: tc.givenParams,
				name:   tc.expect,
			}

			assert.Equal(t, tc.expect, r.Reverse(tc.whenParams...))
		})
	}
}
