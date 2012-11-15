// Package wedge is a package to help cut-down on the oft-written boilerplate code with
// net/http. It is not intended as a framework or a replacement for things like net/http
// or Gorilla. It's a simple module for doing simple things.
package wedge

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	HTML handlertype = iota
	JSON
	STATIC
	ICON
	REDIRECT
	DOWNLOAD
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
type view func(http.ResponseWriter, *http.Request) (string, int)

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

// Helper method which reads a file into memory or returns an error
//
// Used in both Favicon and StaticFiles
func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	// there is only one return but doing it this way means that
	// further additions won't forget to close the fh
	defer file.Close()

	// if we're here, the file exists and we just need to send
	// it to the client.
	b := []string{}
	for {
		reader := make([]byte, FileChunks)
		count, err := file.Read(reader)
		if err != nil {
			return strings.Join(b, ""), nil
		}

		b = append(b, string(reader[:count]))
	}
	panic("Unreachable!")
}
