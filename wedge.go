// Package wedge is a lightweight web framework which intends to
// cut-down on oft written boilerplate code
package wedge

import (
	"net/http"
	"strings"
	"time"
)

const (
	HTML handlertype = iota
	JSON
	STATIC
	ICON
	REDIRECT
)

const (
	FileChunks = 1024
)

var (
	routes  []*url
	TIMEOUT = time.Second
)

// Page handler type
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
