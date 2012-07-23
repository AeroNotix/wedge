package wedge

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	HTTP = iota
	JSON
)

const (
	FileChunks = 1024
)

var (
	routes []*url
)

type handlertype int

type appServer struct {
	port    string
	routes  []*url
	timeout time.Duration
}

// Handler functions should match this signature
type view func(*http.Request) (string, int)

// This is the main 'event loop' for the web server. All requests are
// sent to this handler, which checks the incoming request against
// all the routes we have setup if it finds a match it will invoke
// the handler which is attached to that match.
func (self *appServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	request := req.URL.Path

	for _, route := range self.routes {
		matches := route.match.FindAllStringSubmatch(request, 1)
		if len(matches) > 0 {
			log.Println("Request:", route.name)
			resp, err := route.handler(req)
			if err == 404 {
				http.NotFound(w, req)
				return
			}

			switch route.viewtype {
			case HTTP:
				io.WriteString(w, resp)
				return
			case JSON:
				w.Header().Set("Content-type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"message": resp,
				})
				return
			default:
				panic("Unknown handler type!")
			}
		}
	}
	log.Println("404")
	http.NotFound(w, req)
}

// Basic URL struct which holds a match, a name and a handler function
//
// match:
//     Match is a *regexp.Regexp which we will use to check incoming
//     request URLs against and return the handler on any that match
// name:
//     Name is a simple string of what the url should be referred to as
// handler:
//     Handler is a wedge.view function which we will use against any
//     requests that match `match`.
type url struct {
	match    *regexp.Regexp
	name     string
	handler  view
	viewtype handlertype
}

func (u *url) String() string {
	return fmt.Sprintf(
		"{\n  URL: %s\n  Handler: %s\n}", u.match, u.name,
	)
}

// URL is a function which returns a URL value.
// re:
//     re is a string which will be compiled to a *regexp.Regexp
//     and will panic if the regular expression cannot be compiled
// name:
//     Name is a simple string of what the url should be referred to as
// handler:
//     Handler is a wedge.view function which we will use against any
//     requests that match `match`.
func URL(re, name string, v view, t handlertype) *url {
	match := regexp.MustCompile(re)
	return &url{
		match:    match,
		name:     name,
		handler:  v,
		viewtype: t,
	}
}

// StaticFiles is a not so light wrapper around the URL function
//
// We start off receiving an 'as' string which marks the URL to which
// we match against. We then take a []string which is filepaths to all
// the locations in which an incoming file request should be checked
// against. The file is read in chunks as per the module level constant
// FileChunk.
//
// This function will return a file in a string format ready to be sent
// across the wire.
func StaticFiles(as string, paths ...string) *url {

	return URL(as, "Static File", func(req *http.Request) (string, int) {
		log.Println(req.URL.Path)
		filename := req.URL.Path[len(as):]
		b := []string{}

		for _, path := range paths {
			// Prevent Directory Traversal Attacks
			if len(strings.Split(path, "..")) > 1 {
				return "", http.StatusNotFound
			}

			// Attempt to open the file in using one of the paths
			file, err := os.Open(filepath.Join(path, filename))
			if err != nil {
				continue
			}

			// if we're here, the file exists and we just need to send
			// it to the client.
			for {
				reader := make([]byte, FileChunks)
				count, err := file.Read(reader)
				if err != nil {
					return strings.Join(b, ""), http.StatusOK
				}

				b = append(b, string(reader[:count]))
			}
		}
		return "", http.StatusNotFound
	}, HTTP)

}

// Patterns is a helper function which mutates the global routes map
// by adding a newly created url.
func Patterns(urls ...*url) {
	for _, url := range urls {
		routes = append(routes, url)
	}
}

// Starts the server running on PORT `port` with the timeout duration
func Run(port string, timeout time.Duration) {
	app := &appServer{port, routes, timeout}
	server := http.Server{
		Addr:        ":" + app.port,
		Handler:     app,
		ReadTimeout: app.timeout * time.Second,
	}
	fmt.Printf("Serving on PORT: %s", port)
	fmt.Println("\n")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
