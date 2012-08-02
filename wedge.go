// Package wedge is a lightweight web framework which intends to
// cut-down on oft written boilerplate code
package wedge

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	HTML handlertype = iota
	JSON
	STATIC
	ICON
)

const (
	FileChunks = 1024
)

var (
	routes  []*url
	TIMEOUT = time.Second
)

type handlertype int

// Handler functions should match this signature
type view func(*http.Request) (string, int)

// BasicReplace takes a string and a map[string]string which it uses
// to replace any instances of a key by the value under it.
func BasicReplace(template string, replacement_map map[string]string) string {
	var replacements []string
	for key, val := range replacement_map {
		replacements = append(replacements, key, val)
	}
	replacer := strings.NewReplacer(replacements...)
	return replacer.Replace(template)
}

// Starts the server running on PORT `port` with the timeout duration
func (App *appServer) Run() {
	server := http.Server{
		Addr:        ":" + App.port,
		Handler:     App,
		ReadTimeout: App.timeout * time.Second,
	}
	fmt.Printf("Serving on PORT: %s\n", App.port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
