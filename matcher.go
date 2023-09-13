package pathmatcher

import (
	"sync"
)

// Matcher associates parametrized paths with values.
//
// It's implementation is a light wrapper around tree.go/node struct that manages
// a pool of parameters.
type Matcher[V any] struct {
	tree       *node[V]
	paramsPool sync.Pool
	maxParams  uint
}

func (r *Matcher[V]) getParams() *Params {
	ps, _ := r.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0] // reset slice
	return ps
}

func (r *Matcher[V]) putParams(ps *Params) {
	if ps != nil {
		r.paramsPool.Put(ps)
	}
}

func NewMatcher[V any]() (m *Matcher[V]) {
	m = &Matcher[V]{
		tree: &node[V]{},
		paramsPool: sync.Pool{
			New: func() any {
				ps := make(Params, 0, m.maxParams)
				return &ps
			},
		},
	}
	return m
}

func (m *Matcher[V]) AddPath(path string, value *V) {
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}
	if value == nil {
		panic("value may not be nil")
	}

	m.tree.addPath(path, value)

	m.maxParams = max(m.maxParams, countParams(path))
}

func (m *Matcher[V]) LookupPath(path string) (*V, Params, string, bool) {
	value, params, matchedPath, redir := m.tree.findMatch(path, m.getParams)
	if value == nil {
		m.putParams(params)
		return nil, nil, "", redir
	}
	if params == nil {
		return value, nil, matchedPath, redir
	}
	return value, *params, matchedPath, redir
}
