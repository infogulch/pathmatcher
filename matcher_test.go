package pathmatcher

import (
	"reflect"
	"testing"
)

func TestMatcher(t *testing.T) {
	m := NewMatcher[string]()

	routes := []struct{ path, value string }{
		{"/hello", "world"},
		{"/foo/:bar", "baz"},
	}

	for _, route := range routes {
		m.Add(route.path, route.value)
	}

	checks := []struct {
		path, value, match string
		params             Params
	}{
		{"/hello", "world", "/hello", nil},
		{"/foo/lala", "baz", "/foo/:bar", Params{{"bar", "lala"}}},
		{"/none", "", "", nil},
	}

	for _, check := range checks {
		match, value, params, redir := m.Find(check.path)
		if value != check.value {
			t.Errorf("wrong value returned, expected '%+v', got '%+v'", check.value, value)
		}
		if !reflect.DeepEqual(params, check.params) {
			t.Errorf("got wrong params: expected `%+v`, got `%+v`", check.params, params)
		}
		if match != check.match {
			t.Errorf("wrong path, expected '%s' got '%s'", check.match, match)
		}
		if redir {
			t.Errorf("expected trailing slash flag to be false, got true")
		}
	}
}
