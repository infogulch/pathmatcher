package pathmatcher

import (
	"net/http"
	"testing"
)

func TestHttpMatcher(t *testing.T) {
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
		i := i
		path := "/" + method
		m.AddEndpoint(method, path, &i)
	}

	for i, method := range methods {
		for j, path := range methods {
			path := "/" + path
			v, ps, pt, ts := m.LookupEndpoint(method, path)
			if i != j {
				if v != nil || len(ps) != 0 || pt != "" || ts != false {
					t.Errorf("unexpected match (%s, %s): %d, %+v, %s, %t", method, path, v, ps, pt, ts)
				}
			} else {
				if *v != i || len(ps) != 0 || pt != path || ts != false {
					t.Errorf("incorrect match args (%s, %s): %d, %+v, %s, %t", method, path, v, ps, pt, ts)
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
		i := i
		path := "/" + method
		m.AddEndpoint(method, path, &i)
	}

	tests := []struct{ path, allowed string }{
		{"*", "DELETE, GET, OPTIONS, POST"},
		{"/GET", "GET, OPTIONS"},
	}
	for _, test := range tests {
		allowed := m.Allowed(test.path)
		if allowed != test.allowed {
			t.Errorf("allowed didn't match: expected '%s' got '%s'", test.allowed, allowed)
		}
	}
}
