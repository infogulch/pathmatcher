package pathmatcher

import (
	"sync"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

type Matcher[V any] struct {
	tree       *node[V]
	paramsPool *sync.Pool
	maxParams  *uint
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

func New[V any]() (m *Matcher[V]) {
	m = &Matcher[V]{
		paramsPool: &sync.Pool{
			New: func() any {
				ps := make(Params, 0, *m.maxParams)
				return &ps
			},
		},
		maxParams: new(uint),
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

	*m.maxParams = max(*m.maxParams, countParams(path))
}

func (m *Matcher[V]) LookupPath(path string) (*V, Params, string, bool) {
	if m.tree == nil {
		return nil, nil, "", false
	}
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
