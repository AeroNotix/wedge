// Package wedge is a lightweight web framework which intends to
// cut-down on oft written boilerplate code
package wedge

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	HTML = iota
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

// appServer is our server instance which holds the ServeHTTP method
// so that it satisfies the http.Server interface.
type appServer struct {
	port       string
	routes     []*url
	timeout    time.Duration
	cache_map  *safeMap
	handler404 view
	stat_map   *safeMap
}

// appServer constructor
func NewAppServer(port string, timeout time.Duration) *appServer {
	return &appServer{
		port:      port,
		routes:    make([]*url, 0),
		timeout:   timeout,
		cache_map: NewSafeMap(),
	}
}

// Attaches more *urls to the Routes slice on the appServer value
func (App *appServer) AddURLs(patterns ...*url) {
	for _, url := range patterns {
		App.routes = append(App.routes, url)
	}
}

// EnableStatTracking does exactly what it says on the tin
//
// EnableStatTracking creates a NewSafeMap under the stat_map field which will
// then be used to increment and aggregate hits to URLs.
//
// This function will append a new *url onto the associated appServer. The url
// which this is under is ^/statistics/?$.
func (App *appServer) EnableStatTracking() {
	App.stat_map = NewSafeMap()

	staturl := makeurl("^/statistics/?$", "Statistics", func(req *http.Request) (string, int) {

		rawdata, ok := App.stat_map.Do(func(m map[interface{}]interface{}) interface{} {
			// we could return m here but that would mean we've broken the
			// reason why we made the map safe in the first place.

			outstr := "<table>"
			for key, value := range m {
				outstr += fmt.Sprintf("<tr><td>%s</td>", key.(string))
				outstr += fmt.Sprintf("<td>%d</td></tr>", value.(int))
			}
			outstr += "</table>"
			return outstr
		})

		if !ok {
			return "Failure getting data", 500
		}
		return rawdata.(string), 200

	}, HTML, 0)
	App.routes = append(App.routes, staturl)
}

func (App *appServer) incrementStats(k string) {
	if App.stat_map == nil {
		panic("Cannot increment statistics when it has not been enabled!")
	}
	go App.stat_map.Do(func(m map[interface{}]interface{}) interface{} {
		val, ok := m[k]
		if ok {
			val, ok := val.(int)
			if ok {
				val++
				m[k] = val
			}
		} else {
			m[k] = 1
		}
		return true
	})
}

// Sets the 404 Handler for the appServer to fn.
func (App *appServer) Handler404(fn view) {
	App.handler404 = fn
}

// This is the main 'event loop' for the web server. All requests are
// sent to this handler, which checks the incoming request against
// all the routes we have setup if it finds a match it will invoke
// the handler which is attached to that match.
func (App *appServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	request := req.URL.Path

	for _, route := range App.routes {
		matches := route.match.FindAllStringSubmatch(request, 1)
		if len(matches) > 0 {
			log.Println("Request:", route.name, request)
			if App.stat_map != nil {
				App.incrementStats(route.name)
			}
			resp, status := App.getResponse(route, req)

			if status == 404 {
				App.handle404req(w, req)
				return
			}

			switch route.viewtype {
			case HTML:
				io.WriteString(w, resp)
				return
			case JSON:
				w.Header().Set("Content-type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"message": resp,
				})
				return
			case STATIC:
				if w.Header().Get("Content-Type") == "" {
					reqstr := req.URL.Path[len(route.rawre):]
					ctype := mime.TypeByExtension(filepath.Ext(reqstr))
					w.Header().Set("Content-Type", ctype)
				}
				io.WriteString(w, resp)
				return
			case ICON:
				w.Header().Set("Content-Type", "image/x-icon")
				io.WriteString(w, resp)
				return
			default:
				panic("Unknown handler type!")
			}
		}
	}
	App.handle404req(w, req)
	return
}

// handle404req checks if the 404 handler is a custom one and uses that, if not,
// it uses the built-in NotFound function.
func (App *appServer) handle404req(w http.ResponseWriter, req *http.Request) {
	log.Println("404", req.URL.Path)
	App.incrementStats("404")
	w.WriteHeader(404)

	if App.handler404 != nil {
		resp, _ := App.handler404(req)
		io.WriteString(w, resp)
		return
	} else {
		http.NotFound(w, req)
		return
	}
}

// getResponse checks the *url's cache_duration, if the cache duration
// is zero. Then we never cache the response. Otherwise, we check to
// see if the cache_duration has passed by reading the timeout channel
// if so, we run the URL handler associated with the route and store it's
// new response value. We then store the response in the cache_map and
// return it to the client.
//
// Accessing the cache_map from multiple threads is safe. There are two
// implementations of a safe map included with this library. One is sync'd
// with channels (safeMap) and the other is sync'd with a mutex lock
// (lockMap). We currently use the safeMap.
func (App *appServer) getResponse(route *url, req *http.Request) (string, int) {

	if route.cache_duration == 0 {
		return route.handler(req)
	}

	select {
	case <-route.timeout:
		go func(d time.Duration, ch chan bool) {
			log.Println("Timed out")
			f := time.After(d * TIMEOUT)
			<-f
			go func() {
				ch <- true
			}()
		}(route.cache_duration, route.timeout)
		resp, err := route.handler(req)
		if !App.cache_map.Insert(req.URL.Path, resp) {
			panic("Inserting into cache_map failure!")
		}
		return resp, err
	default:
		resp, ok := App.cache_map.Find(req.URL.Path).(string)
		if !ok {
			resp, _ = route.handler(req)

		}
		if !App.cache_map.Insert(req.URL.Path, resp) {
			panic("Inserting into cache_map failure!")
		}
		return resp, http.StatusOK
	}
	panic("unreachable")
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
	match          *regexp.Regexp
	name           string
	handler        view
	viewtype       handlertype
	rawre          string
	cache_duration time.Duration
	timeout        chan bool
}

func (u *url) String() string {
	return fmt.Sprintf(
		"{\n  URL: %s\n  Handler: %s\n}", u.match, u.name,
	)
}

// Unexported method which forms as the base method to return *url values
//
// We chose to do it like this because we can have specialized methods
// which have a simply API but fill in certain blanks for this. And the
// makeurl method can have a relatively clunky API since the work will
// be done under the hood.
func makeurl(re, name string, v view, t handlertype, duration time.Duration) *url {
	match := regexp.MustCompile(re)
	timeoutchan := make(chan bool)
	if duration > 0 {
		go func() {
			timeoutchan <- true
		}()
	}
	if duration < 0 {
		duration = 30 * 12 * 30 * time.Hour
	}

	return &url{
		match:          match,
		name:           name,
		handler:        v,
		viewtype:       t,
		rawre:          re,
		cache_duration: duration,
		timeout:        timeoutchan,
	}
}

// URL is a function which returns a *url value.
// re:
//     re is a string which will be compiled to a *regexp.Regexp
//     and will panic if the regular expression cannot be compiled
// name:
//     Name is a simple string of what the url should be referred to as
// handler:
//     Handler is a wedge.view function which we will use against any
//     requests that match `match`.
func URL(re, name string, v view, t handlertype) *url {
	return makeurl(re, name, v, t, 0)
}

// StaticFiles is a not so light wrapper around the URL function
//
// We start off receiving an 'as' string which marks the URL to which
// we match against. We then take a []string which is filepaths to all
// the locations in which an incoming file request should be checked
// against. The file is read in chunks as per the module level constant4
// FileChunk.
//
// This function will return a file in a string format ready to be sent
// across the wire.
func StaticFiles(as string, paths ...string) *url {
	return makeurl(as, "Static File", func(req *http.Request) (string, int) {
		filename := req.URL.Path[len(as):]
		for _, path := range paths {
			// Prevent Directory Traversal Attacks
			if len(strings.Split(path, "..")) > 1 {
				return "", http.StatusNotFound
			}
			out_data, err := readFile(filepath.Join(path, filename))
			if err != nil {
				continue
			}
			return out_data, http.StatusOK
		}
		return "", http.StatusNotFound
	}, STATIC, -1)
}

// CacheURL returns a URL which has caching enabled for time.Duration d.
func CacheURL(re, name string, v view, t handlertype, d time.Duration) *url {
	return makeurl(re, name, v, t, d)
}

// Favicon takes a path to some file which you want to be returned when
// a request comes through for ^/favicon.ico$. By default this will cache
// for TIMEOUT * 10.
func Favicon(path string) *url {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	file.Close()

	return makeurl("^/favicon.ico$", "Favicon",
		func(req *http.Request) (string, int) {
			out_data, err := readFile(path)
			if err != nil {
				return "", http.StatusNotFound
			}
			return out_data, http.StatusOK
		}, ICON, -1)
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
