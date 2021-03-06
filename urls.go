package wedge

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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

	// Initialize the channel and seed with a value
	// so the first request will put the response
	// into memory
	if duration < 0 {
		duration = 30 * 12 * 30 * time.Hour
	}
	if duration > 0 {
		go func() {
			timeoutchan <- true
		}()
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

// Download is a function which returns a *url value.
//
// This function simply gives access to the correct content header types
// so a file is downloaded instead of displayed.
func Download(re, name string, v view) *url {
	return makeurl(re, name, v, DOWNLOAD, 0)
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
	return makeurl(as, "Static File", func(w http.ResponseWriter, req *http.Request) (string, int) {
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
		func(w http.ResponseWriter, req *http.Request) (string, int) {
			out_data, err := readFile(path)
			if err != nil {
				return "", http.StatusNotFound
			}
			return out_data, http.StatusOK
		}, ICON, -1)
}

// Redirect is a simple method of allowing paths to be redirected to other URLs.
func Redirect(path, to string, code int) *url {
	return makeurl(path, fmt.Sprintf("Redirecting %s => %s", path, to),
		func(w http.ResponseWriter, req *http.Request) (string, int) {
			return to, code
		}, REDIRECT, 0)
}

// Returns data as the robots.txt file
func Robots(data string) *url {
	return makeurl("^/robots.txt$", "Attack of the robots...robots.txt",
		func(w http.ResponseWriter, req *http.Request) (string, int) {
			return data, http.StatusOK
		}, HTML, -1)
}
