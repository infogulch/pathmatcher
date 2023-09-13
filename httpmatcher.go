package pathmatcher

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
)

// HttpMatcher associates endpoints (methods + parameterized paths) with values.
//
// Implemented as a map of method names to Matchers with a shared param pool.
type HttpMatcher[V any] struct {
	trees map[string]*node[V]

	paramsPool sync.Pool
	maxParams  uint
}

func (r *HttpMatcher[V]) getParams() *Params {
	ps, _ := r.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0] // reset slice
	return ps
}

func (r *HttpMatcher[V]) putParams(ps *Params) {
	if ps != nil {
		r.paramsPool.Put(ps)
	}
}

func NewHttpMatcher[V any]() (m *HttpMatcher[V]) {
	m = &HttpMatcher[V]{
		trees: make(map[string]*node[V]),
		paramsPool: sync.Pool{
			New: func() any {
				ps := make(Params, 0, m.maxParams)
				return &ps
			},
		},
	}
	return
}

var validMethods map[string]struct{} = map[string]struct{}{
	"GET":     {},
	"HEAD":    {},
	"POST":    {},
	"PUT":     {},
	"PATCH":   {},
	"DELETE":  {},
	"CONNECT": {},
	"OPTIONS": {},
	"TRACE":   {},
}

func (m *HttpMatcher[V]) AddEndpoint(method, path string, value *V) {
	if _, ok := validMethods[method]; !ok {
		panic(fmt.Sprintf("invalid method '%s'", method))
	}

	tree, ok := m.trees[method]
	if !ok {
		tree = &node[V]{}
		m.trees[method] = tree
	}

	tree.addPath(path, value)
	m.maxParams = max(m.maxParams, countParams(path))
}

func (m *HttpMatcher[V]) LookupEndpoint(method, path string) (*V, Params, string, bool) {
	tree, ok := m.trees[method]
	if !ok {
		return nil, nil, "", false
	}
	value, params, matchedPath, redir := tree.findMatch(path, m.getParams)
	if value == nil {
		m.putParams(params)
		return nil, nil, "", redir
	}
	if params == nil {
		return value, nil, matchedPath, redir
	}
	return value, *params, matchedPath, redir
}

// Allowed returns an Allow list [1] based on the methods and endpoints set in
// the matcher.
//
// [1]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Allow
func (m *HttpMatcher[V]) Allowed(path string) string {
	allowedList := (&[9]string{http.MethodOptions})[:1]
	if path == "*" {
		for method := range m.trees {
			if method == http.MethodOptions {
				continue
			}
			allowedList = append(allowedList, method)
		}
	} else {
		for method := range m.trees {
			if method == http.MethodOptions {
				continue
			}
			value, ps, _, _ := m.trees[method].findMatch(path, m.getParams)
			m.putParams(ps)
			if value != nil {
				allowedList = append(allowedList, method)
			}
		}
	}
	slices.Sort(allowedList)
	return strings.Join(allowedList, ", ")
}
