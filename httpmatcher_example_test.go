package pathmatcher_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/infogulch/pathmatcher"
)

func Index(w http.ResponseWriter, r *http.Request, _ pathmatcher.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps pathmatcher.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

type Handler func(w http.ResponseWriter, r *http.Request, ps pathmatcher.Params)

func Example() {
	matcher := pathmatcher.NewHttpMatcher[Handler]()
	matcher.GET("/", Index)
	matcher.GET("/hello/:name", Hello)

	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, handler, params, _ := matcher.Find(r.Method, r.URL.Path)
		if handler != nil {
			handler(w, r, params)
		} else {
			http.NotFound(w, r)
		}
	})))
}
