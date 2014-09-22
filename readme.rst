WEDGE IS DEPRECATED AND IS NO LONGER BEING DEVELOPED

LPT: DO NOT USE. LEFT HERE AS A MONUMENT TO MY SINS.

Wedge
=====

Wedge is intended to cut-down on the boilerplate when creating dead simple webapps. There's no super
over-the-top functionality in the majority of webapps I've seen yet we all write the same boring code.

Wedge, for the time being, will allow you to write simple functions which do simple things. You need a
function on a URL? No problem. You want to easily write a simple JSON response for a URL? No problem!
You want to have a multi-tiered RPC cluster with flash failover support and other such magic? Use the
standard library and write it yourself, Wedge would not be a good fit.


Features:

* Easily set up functions to hang off URLs.
* Static files retrieval on any URL, cached.
* Cache URLs permanently or for a specified time.
* Form processing.
* Custom 404/500 handlers.
* Statistics tracking.
* Not much else.

Usage:

.. code-block:: go

    // Main page
    func Index(w http.ResponseWriter, req *http.Request) (string, int) {
        return "Hello world!", http.StatusOK
    }

    func Page404(w http.ResponseWriter, req *http.Request) (string, int) {
    	return "An oopsie!", http.StatusNotFound
    }

    func main() {

    	App := wedge.NewAppServer(12345, 30)
    	App.AddURLs(
    		wedge.Favicon(filepath.Join(DIRNAME, "static", "favicon.ico")),
    		wedge.StaticFiles("/static/", filepath.Join(DIRNAME, "static/")),
    		wedge.URL("^/awesome/$", "Getting awesome", Awsum, wedge.HTML),
    		wedge.CacheURL("^/$", "Index", Index, wedge.HTML, -1),
    	)
    	App.Handler404(Page404)
    	App.EnableStatTracking()      // stat tracking on ^/statistics/?$
    	App.Run()
    }
