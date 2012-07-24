package wedge

import (
	"fmt"
	"net/http"
	"regexp"
	"log"
	"time"
	"io"
	"encoding/json"
)

const  (
	HTTP = iota
	JSON
)

type handlertype int

type appServer struct {
	port string
	routes []*url
	timeout time.Duration
}

// Handler functions should match this signature
type view func(*http.Request) string

// This is the main 'event loop' for the web server. All requests are
// sent to this handler, which checks the incoming request against
// all the routes we have setup if it finds a match it will invoke
// the handler which is attached to that match.
func (self *appServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	request := req.URL.Path
	
	for _, route := range self.routes {
		matches := route.match.FindAllStringSubmatch(request, 1)
		if len(matches) > 0 {
			log.Printf("Request on: %s Handled by: %s", route.rawre, route.name)
			resp := route.handler(req)

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
	http.NotFound(w, req)
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
func URL(re, name string, v view, t handlertype)  *url {
	match := regexp.MustCompile(re)
	return &url{
		match: match,
		name: name,
		handler: v,
		viewtype: t,
		rawre: re,
	}
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
	match *regexp.Regexp
	name string
	handler view
	viewtype handlertype
	rawre string
}

func (u *url) String() string {
	return fmt.Sprintf(
		"{\n  URL: %s\n  Handler: %s\n}", u.match, u.name,
	)
}

// Patterns is a helper function which returns a *[]*url.
func Patterns(urls ...*url) (*[]*url) {
	r := make([]*url, 0)
	for _, url := range urls {
		r = append(r, url)
	}

	return &r
}

// Starts the server running on PORT `port` with the timeout duration
func Run(patterns *[]*url, port string, timeout time.Duration) {
	app := &appServer{port, *patterns, timeout}
	server := http.Server{
		Addr: ":"+app.port,
		Handler: app,
		ReadTimeout: app.timeout * time.Second, 
	}
	fmt.Println(fmt.Sprintf("Serving on PORT: %s", port))
	server.ListenAndServe()
}