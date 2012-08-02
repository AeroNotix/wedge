package wedge

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"time"
)

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

		rawdata, ok := App.stat_map.Do(func(m freemap) interface{} {
			// we could return m here but that would mean we've broken the
			// reason why we made the map safe in the first place.
			outstr := `<!DOCTYPE html><html><table border="2">`
			outstr += `<tr><th>URL</th><th>Hits</th></tr>`
			var urllist []string
			for key, _ := range m {
				urllist = append(urllist, key.(string))
			}
			sort.Strings(urllist)
			for _, key := range urllist {
				outstr += fmt.Sprintf("<tr><td>%s</td>", key)
				outstr += fmt.Sprintf("<td>%d</td></tr>", m[key].(int))
			}
			outstr += `</table>
					   </html>`
			return outstr
		})

		if !ok {
			return "Failure getting data", 500
		}
		return rawdata.(string), 200

	}, HTML, 0)
	App.routes = append(App.routes, staturl)
}

// incrementStats is a non-blocking method to increment a page counter
// for individual routes.
func (App *appServer) incrementStats(k string) {
	if App.stat_map == nil {
		panic("Cannot increment statistics when it has not been enabled!")
	}

	// create a goroutine which sends a function literal to the async
	// map which tries to increment the value under the k string.
	go App.stat_map.Do(func(m freemap) interface{} {
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
//
// If somehow the URL it finds has been created with a non-existant
// handler type it will panic.
func (App *appServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	request := req.URL.Path

	for _, route := range App.routes {
		matches := route.match.FindAllStringSubmatch(request, 1)
		if len(matches) > 0 {
			log.Println("Request:", route.name, request)

			if App.stat_map != nil {
				App.incrementStats(request)
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
	log.Println("404 on path:", req.URL.Path)

	if App.stat_map != nil {
		App.incrementStats("404" + req.URL.Path)
	}

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
		// reset the timeout timer
		go func(d time.Duration, ch chan bool) {
			log.Println("Timed out")
			f := time.After(d * TIMEOUT)
			<-f
			go func() {
				ch <- true
			}()
		}(route.cache_duration, route.timeout)
		// get the new response and cache it in the map
		resp, err := route.handler(req)
		if !App.cache_map.Insert(req.URL.Path, resp) {
			panic("Inserting into cache_map failure!")
		}
		return resp, err
	default:
		resp, ok := App.cache_map.Find(req.URL.Path).(string)
		var status int
		if !ok {
			resp, status = route.handler(req)
		}
		if status != 404 {
			if !App.cache_map.Insert(req.URL.Path, resp) {
				panic("Inserting into cache_map failure!")
			}
		}
		return resp, status
	}
	panic("unreachable")
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
