package pathmatcher

import "testing"

func TestMatcher(t *testing.T) {
	m := NewMatcher[string]()

	a := "world"
	m.AddPath("/hello", &a)
	b := "baz"
	m.AddPath("/foo/:bar", &b)

	{
		v, ps, pt, ts := m.LookupPath("/hello")
		if v != &a {
			t.Errorf("wrong value returned, expected %+v, got %+v", a, v)
		}
		if len(ps) != 0 {
			t.Errorf("got params, expected none %+v", ps)
		}
		if pt != "/hello" {
			t.Errorf("wrong path, expected '%s' got '%s'", "/hello", pt)
		}
		if ts {
			t.Errorf("expected trailing slash flag to be false, got true")
		}
	}

	{
		v, ps, pt, ts := m.LookupPath("/foo/lala")
		if v != &b {
			t.Errorf("wrong value returned, expected %+v, got %+v", a, v)
		}
		if len(ps) != 1 || ps[0].Key != "bar" || ps[0].Value != "lala" {
			t.Errorf("wrong params, expected ['lala'], got %+v", ps)
		}
		if pt != "/foo/:bar" {
			t.Errorf("wrong path, expected '%s' got '%s'", "/foo/:bar", pt)
		}
		if ts {
			t.Errorf("expected trailing slash flag to be false, got true")
		}
	}

	{
		v, ps, pt, ts := m.LookupPath("/none")
		if v != nil {
			t.Errorf("wrong value returned, expected nil, got %+v", v)
		}
		if len(ps) != 0 {
			t.Errorf("wrong params, expected none, got %+v", ps)
		}
		if pt != "" {
			t.Errorf("wrong path, expected '%s' got '%s'", "/foo/:bar", pt)
		}
		if ts {
			t.Errorf("expected trailing slash flag to be false, got true")
		}
	}
}
