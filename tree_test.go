// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package pathmatcher

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// func printChildren(n *node, prefix string) {
// 	fmt.Printf(" %02d %s%s[%d] %v %t %d \r\n", n.priority, prefix, n.path, len(n.children), n.handle, n.wildChild, n.nType)
// 	for l := len(n.path); l > 0; l-- {
// 		prefix += " "
// 	}
// 	for _, child := range n.children {
// 		printChildren(child, prefix)
// 	}
// }

type testRequests []struct {
	path        string
	shouldMatch bool
	value       int
	matchedPath string
	params      Params
}

func getParams() *Params {
	ps := make(Params, 0, 20)
	return &ps
}

func checkRequests(t *testing.T, tree *node[int], requests testRequests) {
	for _, request := range requests {
		value, psp, matchedPath, _ := tree.findMatch(request.path, getParams)

		switch {
		case value == nil && !request.shouldMatch:
		case value == nil && request.shouldMatch:
			t.Errorf("value mismatch for path '%s': Expected non-nil value", request.path)
		case value != nil && !request.shouldMatch:
			t.Errorf("value mismatch for path '%s': Expected nil value, got %d", request.path, *value)
		case value != nil && request.shouldMatch:
			if *value != request.value {
				t.Errorf("value mismatch for path '%s': Expected %d, got %d", request.path, request.value, *value)
			}
			if matchedPath != request.matchedPath {
				t.Errorf("path mismatch for '%s': Matched '%s', expected to match '%s' ", request.path, matchedPath, request.matchedPath)
			}
		}

		var ps Params
		if psp != nil {
			ps = *psp
		}

		if !reflect.DeepEqual(ps, request.params) {
			t.Errorf("params mismatch for route '%s': expected `%+v`, got `%+v`", request.path, request.params, ps)
		}
	}
}

func checkPriorities[T any](t *testing.T, n *node[T]) uint32 {
	var prio uint32
	for i := range n.children {
		prio += checkPriorities(t, n.children[i])
	}

	if n.value != nil {
		prio++
	}

	if n.priority != prio {
		t.Errorf(
			"priority mismatch for node '%s': is %d, should be %d",
			n.path, n.priority, prio,
		)
	}

	return prio
}

func TestCountParams(t *testing.T) {
	if countParams("/path/:param1/static/*catch-all") != 2 {
		t.Fail()
	}
	if countParams(strings.Repeat("/:param", 256)) != 256 {
		t.Fail()
	}
}

func TestTreeAddAndGet(t *testing.T) {
	tree := &node[int]{}

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}
	for i, route := range routes {
		i := i
		tree.addPath(route, &i)
	}

	// printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/a", true, 4, "/a", nil},
		{"/", false, -1, "", nil},
		{"/hi", true, 0, "/hi", nil},
		{"/contact", true, 1, "/contact", nil},
		{"/co", true, 2, "/co", nil},
		{"/con", false, -1, "", nil},  // key mismatch
		{"/cona", false, -1, "", nil}, // key mismatch
		{"/no", false, -1, "", nil},   // no matching child
		{"/ab", true, 5, "/ab", nil},
		{"/α", true, 9, "/α", nil},
		{"/β", true, 10, "/β", nil},
	})

	checkPriorities(t, tree)
}

func TestTreeWildcard(t *testing.T) {
	tree := &node[int]{}

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/user_:name",
		"/user_:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
	}
	for i, route := range routes {
		i := i
		tree.addPath(route, &i)
	}

	// printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", true, 0, "/", nil},
		{"/cmd/test/", true, 2, "/cmd/:tool/", Params{Param{"tool", "test"}}},
		{"/cmd/test", false, -1, "", nil},
		{"/cmd/test/3", true, 1, "/cmd/:tool/:sub", Params{Param{"tool", "test"}, Param{"sub", "3"}}},
		{"/src/", true, 3, "/src/*filepath", Params{Param{"filepath", "/"}}},
		{"/src/some/file.png", true, 3, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
		{"/search/", true, 4, "/search/", nil},
		{"/search/someth!ng+in+ünìcodé", true, 5, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/search/someth!ng+in+ünìcodé/", false, 5, "", nil},
		{"/user_gopher", true, 6, "/user_:name", Params{Param{"name", "gopher"}}},
		{"/user_gopher/about", true, 7, "/user_:name/about", Params{Param{"name", "gopher"}}},
		{"/files/js/inc/framework.js", true, 8, "/files/:dir/*filepath", Params{Param{"dir", "js"}, Param{"filepath", "/inc/framework.js"}}},
		{"/info/gordon/public", true, 12, "/info/:user/public", Params{Param{"user", "gordon"}}},
		{"/info/gordon/project/go", true, 13, "/info/:user/project/:project", Params{Param{"user", "gordon"}, Param{"project", "go"}}},
	})

	checkPriorities(t, tree)
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

type testRoute struct {
	path     string
	conflict bool
}

func testRoutes(t *testing.T, routes []testRoute) {
	tree := &node[int]{}

	for i := range routes {
		route := routes[i]
		recv := catchPanic(func() {
			tree.addPath(route.path, nil)
		})

		if route.conflict {
			if recv == nil {
				t.Errorf("no panic for conflicting route '%s'", route.path)
			}
		} else if recv != nil {
			t.Errorf("unexpected panic for route '%s': %v", route.path, recv)
		}
	}

	// printChildren(tree, "")
}

func TestTreeWildcardConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/:tool/:sub", false},
		{"/cmd/vet", true},
		{"/src/*filepath", false},
		{"/src/*filepathx", true},
		{"/src/", true},
		{"/src1/", false},
		{"/src1/*filepath", true},
		{"/src2*filepath", true},
		{"/search/:query", false},
		{"/search/invalid", true},
		{"/user_:name", false},
		{"/user_x", true},
		{"/user_:name", false},
		{"/id:id", false},
		{"/id/:id", true},
	}
	testRoutes(t, routes)
}

func TestTreeChildConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/vet", false},
		{"/cmd/:tool/:sub", true},
		{"/src/AUTHORS", false},
		{"/src/*filepath", true},
		{"/user_x", false},
		{"/user_:name", true},
		{"/id/:id", false},
		{"/id:id", true},
		{"/:id", true},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeDupliatePath(t *testing.T) {
	tree := &node[int]{}

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}
	for i := range routes {
		i := i
		route := routes[i]
		recv := catchPanic(func() {
			tree.addPath(route, &i)
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", route, recv)
		}

		// Add again
		recv = catchPanic(func() {
			tree.addPath(route, nil)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting duplicate route '%s", route)
		}
	}

	// printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", true, 0, "/", nil},
		{"/doc/", true, 1, "/doc/", nil},
		{"/src/some/file.png", true, 2, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
		{"/search/someth!ng+in+ünìcodé", true, 3, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", true, 4, "/user_:name", Params{Param{"name", "gopher"}}},
	})
}

func TestEmptyWildcardName(t *testing.T) {
	tree := &node[int]{}

	routes := [...]string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}
	for i, route := range routes {
		i := i
		recv := catchPanic(func() {
			tree.addPath(route, &i)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting route with empty wildcard name '%s", route)
		}
	}
}

func TestTreeCatchAllConflict(t *testing.T) {
	routes := []testRoute{
		{"/src/*filepath/x", true},
		{"/src2/", false},
		{"/src2/*filepath/x", true},
		{"/src3/*filepath", false},
		{"/src3/*filepath/x", true},
	}
	testRoutes(t, routes)
}

func TestTreeCatchAllConflictRoot(t *testing.T) {
	routes := []testRoute{
		{"/", false},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeCatchMaxParams(t *testing.T) {
	tree := &node[int]{}
	var route = "/cmd/*filepath"
	var id int = 0
	tree.addPath(route, &id)
}

func TestTreeDoubleWildcard(t *testing.T) {
	const panicMsg = "only one wildcard per path segment is allowed"

	routes := [...]string{
		"/:foo:bar",
		"/:foo:bar/",
		"/:foo*bar",
	}

	for i, route := range routes {
		i := i
		tree := &node[int]{}
		recv := catchPanic(func() {
			tree.addPath(route, &i)
		})

		if rs, ok := recv.(string); !ok || !strings.HasPrefix(rs, panicMsg) {
			t.Fatalf(`"Expected panic "%s" for route '%s', got "%v"`, panicMsg, route, recv)
		}
	}
}

func TestTreeTrailingSlashRedirect(t *testing.T) {
	tree := &node[int]{}

	routes := [...]string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/admin",
		"/admin/:category",
		"/admin/:category/:page",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
		"/api/hello/:name",
		"/vendor/:x/*y",
	}
	for i, route := range routes {
		i := i
		recv := catchPanic(func() {
			tree.addPath(route, &i)
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", route, recv)
		}
	}

	// printChildren(tree, "")

	tsrRoutes := [...]string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/src",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/admin/",
		"/admin/config/",
		"/admin/config/permissions/",
		"/doc/",
		"/vendor/x",
	}
	for _, route := range tsrRoutes {
		handler, _, _, tsr := tree.findMatch(route, nil)
		if handler != nil {
			t.Fatalf("non-nil handler for TSR route '%s", route)
		} else if !tsr {
			t.Errorf("expected TSR recommendation for route '%s'", route)
		}
	}

	noTsrRoutes := [...]string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
		"/api/world/abc",
	}
	for _, route := range noTsrRoutes {
		handler, _, _, tsr := tree.findMatch(route, nil)
		if handler != nil {
			t.Fatalf("non-nil handler for No-TSR route '%s", route)
		} else if tsr {
			t.Errorf("expected no TSR recommendation for route '%s'", route)
		}
	}
}

func TestTreeRootTrailingSlashRedirect(t *testing.T) {
	tree := &node[int]{}
	n := 0
	recv := catchPanic(func() {
		tree.addPath("/:test", &n)
	})
	if recv != nil {
		t.Fatalf("panic inserting test route: %v", recv)
	}

	handler, _, _, tsr := tree.findMatch("/", nil)
	if handler != nil {
		t.Fatalf("non-nil handler")
	} else if tsr {
		t.Errorf("expected no TSR recommendation")
	}
}

func TestTreeFindCaseInsensitivePath(t *testing.T) {
	tree := &node[int]{}

	longPath := "/l" + strings.Repeat("o", 128) + "ng"
	lOngPath := "/l" + strings.Repeat("O", 128) + "ng/"

	routes := [...]string{
		"/hi",
		"/b/",
		"/ABC/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/doc/go/away",
		"/no/a",
		"/no/b",
		"/Π",
		"/u/apfêl/",
		"/u/äpfêl/",
		"/u/öpfêl",
		"/v/Äpfêl/",
		"/v/Öpfêl",
		"/w/♬",  // 3 byte
		"/w/♭/", // 3 byte, last byte differs
		"/w/𠜎",  // 4 byte
		"/w/𠜏/", // 4 byte
		longPath,
	}

	for i, route := range routes {
		i := i
		recv := catchPanic(func() {
			tree.addPath(route, &i)
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", route, recv)
		}
	}

	// Check out == in for all registered routes
	// With fixTrailingSlash = true
	for i := range routes {
		route := routes[i]
		out, found := tree.findCaseInsensitivePath(route, true)
		if !found {
			t.Errorf("Route '%s' not found!", route)
		} else if out != route {
			t.Errorf("Wrong result for route '%s': %s", route, out)
		}
	}
	// With fixTrailingSlash = false
	for i := range routes {
		route := routes[i]
		out, found := tree.findCaseInsensitivePath(route, false)
		if !found {
			t.Errorf("Route '%s' not found!", route)
		} else if out != route {
			t.Errorf("Wrong result for route '%s': %s", route, out)
		}
	}

	tests := []struct {
		in    string
		out   string
		found bool
		slash bool
	}{
		{"/HI", "/hi", true, false},
		{"/HI/", "/hi", true, true},
		{"/B", "/b/", true, true},
		{"/B/", "/b/", true, false},
		{"/abc", "/ABC/", true, true},
		{"/abc/", "/ABC/", true, false},
		{"/aBc", "/ABC/", true, true},
		{"/aBc/", "/ABC/", true, false},
		{"/abC", "/ABC/", true, true},
		{"/abC/", "/ABC/", true, false},
		{"/SEARCH/QUERY", "/search/QUERY", true, false},
		{"/SEARCH/QUERY/", "/search/QUERY", true, true},
		{"/CMD/TOOL/", "/cmd/TOOL/", true, false},
		{"/CMD/TOOL", "/cmd/TOOL/", true, true},
		{"/SRC/FILE/PATH", "/src/FILE/PATH", true, false},
		{"/x/Y", "/x/y", true, false},
		{"/x/Y/", "/x/y", true, true},
		{"/X/y", "/x/y", true, false},
		{"/X/y/", "/x/y", true, true},
		{"/X/Y", "/x/y", true, false},
		{"/X/Y/", "/x/y", true, true},
		{"/Y/", "/y/", true, false},
		{"/Y", "/y/", true, true},
		{"/Y/z", "/y/z", true, false},
		{"/Y/z/", "/y/z", true, true},
		{"/Y/Z", "/y/z", true, false},
		{"/Y/Z/", "/y/z", true, true},
		{"/y/Z", "/y/z", true, false},
		{"/y/Z/", "/y/z", true, true},
		{"/Aa", "/aa", true, false},
		{"/Aa/", "/aa", true, true},
		{"/AA", "/aa", true, false},
		{"/AA/", "/aa", true, true},
		{"/aA", "/aa", true, false},
		{"/aA/", "/aa", true, true},
		{"/A/", "/a/", true, false},
		{"/A", "/a/", true, true},
		{"/DOC", "/doc", true, false},
		{"/DOC/", "/doc", true, true},
		{"/NO", "", false, true},
		{"/DOC/GO", "", false, true},
		{"/π", "/Π", true, false},
		{"/π/", "/Π", true, true},
		{"/u/ÄPFÊL/", "/u/äpfêl/", true, false},
		{"/u/ÄPFÊL", "/u/äpfêl/", true, true},
		{"/u/ÖPFÊL/", "/u/öpfêl", true, true},
		{"/u/ÖPFÊL", "/u/öpfêl", true, false},
		{"/v/äpfêL/", "/v/Äpfêl/", true, false},
		{"/v/äpfêL", "/v/Äpfêl/", true, true},
		{"/v/öpfêL/", "/v/Öpfêl", true, true},
		{"/v/öpfêL", "/v/Öpfêl", true, false},
		{"/w/♬/", "/w/♬", true, true},
		{"/w/♭", "/w/♭/", true, true},
		{"/w/𠜎/", "/w/𠜎", true, true},
		{"/w/𠜏", "/w/𠜏/", true, true},
		{lOngPath, longPath, true, true},
	}
	// With fixTrailingSlash = true
	for _, test := range tests {
		out, found := tree.findCaseInsensitivePath(test.in, true)
		if found != test.found || (found && (out != test.out)) {
			t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
				test.in, out, found, test.out, test.found)
			return
		}
	}
	// With fixTrailingSlash = false
	for _, test := range tests {
		out, found := tree.findCaseInsensitivePath(test.in, false)
		if test.slash {
			if found { // test needs a trailingSlash fix. It must not be found!
				t.Errorf("Found without fixTrailingSlash: %s; got %s", test.in, out)
			}
		} else {
			if found != test.found || (found && (out != test.out)) {
				t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
					test.in, out, found, test.out, test.found)
				return
			}
		}
	}
}

func TestTreeInvalidNodeType(t *testing.T) {
	const panicMsg = "invalid node type"

	tree := &node[int]{}
	a, b := 0, 1
	tree.addPath("/", &a)
	tree.addPath("/:page", &b)

	// set invalid node type
	tree.children[0].nType = 42

	// normal lookup
	recv := catchPanic(func() {
		tree.findMatch("/test", nil)
	})
	if rs, ok := recv.(string); !ok || rs != panicMsg {
		t.Fatalf("Expected panic '"+panicMsg+"', got '%v'", recv)
	}

	// case-insensitive lookup
	recv = catchPanic(func() {
		tree.findCaseInsensitivePath("/test", true)
	})
	if rs, ok := recv.(string); !ok || rs != panicMsg {
		t.Fatalf("Expected panic '"+panicMsg+"', got '%v'", recv)
	}
}

func TestTreeWildcardConflictEx(t *testing.T) {
	conflicts := [...]struct {
		route        string
		segPath      string
		existPath    string
		existSegPath string
	}{
		{"/who/are/foo", "/foo", `/who/are/\*you`, `/\*you`},
		{"/who/are/foo/", "/foo/", `/who/are/\*you`, `/\*you`},
		{"/who/are/foo/bar", "/foo/bar", `/who/are/\*you`, `/\*you`},
		{"/conxxx", "xxx", `/con:tact`, `:tact`},
		{"/conooo/xxx", "ooo", `/con:tact`, `:tact`},
	}

	for i, conflict := range conflicts {
		i := i
		// I have to re-create a 'tree', because the 'tree' will be
		// in an inconsistent state when the loop recovers from the
		// panic which threw by 'addRoute' function.
		tree := &node[int]{}
		routes := [...]string{
			"/con:tact",
			"/who/are/*you",
			"/who/foo/hello",
		}

		for j, route := range routes {
			j := j
			tree.addPath(route, &j)
		}

		recv := catchPanic(func() {
			tree.addPath(conflict.route, &i)
		})

		if !regexp.MustCompile(fmt.Sprintf("'%s' in new path .* conflicts with existing wildcard '%s' in existing prefix '%s'", conflict.segPath, conflict.existSegPath, conflict.existPath)).MatchString(fmt.Sprint(recv)) {
			t.Fatalf("invalid wildcard conflict error (%v)", recv)
		}
	}
}

func TestRedirectTrailingSlash(t *testing.T) {
	var data = []struct {
		path string
	}{
		{"/hello/:name"},
		{"/hello/:name/123"},
		{"/hello/:name/234"},
	}

	node := &node[int]{}
	for i, item := range data {
		i := i
		node.addPath(item.path, &i)
	}

	_, _, _, tsr := node.findMatch("/hello/abx/", nil)
	if tsr != true {
		t.Fatalf("want true, is false")
	}
}
