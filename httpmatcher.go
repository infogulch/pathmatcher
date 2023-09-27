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

func methodValid(method string) bool {
	for _, m := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE"} {
		if method == m {
			return true
		}
	}
	return false
}

func (m *HttpMatcher[V]) Add(method, path string, value V) {
	if !methodValid(method) {
		panic(fmt.Sprintf("invalid method '%s'", method))
	}

	tree, ok := m.trees[method]
	if !ok {
		tree = &node[V]{}
		m.trees[method] = tree
	}

	tree.addPath(path, &value)
	m.maxParams = max(m.maxParams, countParams(path))
}

func (m *HttpMatcher[V]) GET(path string, value V)     { m.Add(http.MethodGet, path, value) }
func (m *HttpMatcher[V]) HEAD(path string, value V)    { m.Add(http.MethodHead, path, value) }
func (m *HttpMatcher[V]) POST(path string, value V)    { m.Add(http.MethodPost, path, value) }
func (m *HttpMatcher[V]) PUT(path string, value V)     { m.Add(http.MethodPut, path, value) }
func (m *HttpMatcher[V]) PATCH(path string, value V)   { m.Add(http.MethodPatch, path, value) }
func (m *HttpMatcher[V]) DELETE(path string, value V)  { m.Add(http.MethodDelete, path, value) }
func (m *HttpMatcher[V]) TRACE(path string, value V)   { m.Add(http.MethodTrace, path, value) }
func (m *HttpMatcher[V]) CONNECT(path string, value V) { m.Add(http.MethodConnect, path, value) }
func (m *HttpMatcher[V]) OPTIONS(path string, value V) { m.Add(http.MethodOptions, path, value) }

func (m *HttpMatcher[V]) Find(method, path string) (match string, value V, params Params, redir bool) {
	tree, ok := m.trees[method]
	if !ok {
		return
	}
	var pvalue *V
	var pparams *Params
	pvalue, pparams, match, redir = tree.findMatch(path, m.getParams)
	if pvalue == nil {
		m.putParams(pparams)
		return
	}
	if pparams != nil {
		params = *pparams
	}
	value = *pvalue
	return
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
