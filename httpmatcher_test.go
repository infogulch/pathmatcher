package pathmatcher

import (
	"net/http"
	"testing"
)

func TestHttpMatcherMethods(t *testing.T) {
	m := NewHttpMatcher[int]()

	methods := [...]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}
	for i, method := range methods {
		path := "/" + method
		m.Add(method, path, i)
	}

	// Try every combination of `METHOD /PATH`, only i==j should match.
	for i, method := range methods {
		for j, path := range methods {
			path := "/" + path
			match, value, params, redir := m.Find(method, path)
			if i != j {
				if value != 0 || len(params) != 0 || match != "" || redir != false {
					t.Errorf("unexpected match (%s, %s): %d, %+v, %s, %t", method, path, value, params, match, redir)
				}
			} else {
				if value != i || len(params) != 0 || match != path || redir != false {
					t.Errorf("incorrect match args (%s, %s): %d, %+v, %s, %t", method, path, value, params, match, redir)
				}
			}
		}
	}
}

func TestAllowed(t *testing.T) {
	m := NewHttpMatcher[int]()

	methods := [...]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodDelete,
	}
	for i, method := range methods {
		path := "/" + method
		m.Add(method, path, i)
	}

	tests := []struct{ path, allowed string }{
		{"*", "DELETE, GET, OPTIONS, POST"},
		{"/GET", "GET, OPTIONS"},
		{"/foo", "OPTIONS"},
	}
	for _, test := range tests {
		allowed := m.Allowed(test.path)
		if allowed != test.allowed {
			t.Errorf("allowed didn't match: expected '%s' got '%s'", test.allowed, allowed)
		}
	}
}
