Wedge
=====

Wedge is intended to cut-down on the boilerplate when creating dead simple webapps. There's no super
over-the-top functionality in the majority of webapps I've seen yet we all write the same boring code.

Wedge, for the time being, will allow you to write simple functions which do simple things. You need a
function on a URL? No problem. You want to easily write a simple JSON response for a URL? No problem!
You want to have a multi-tiered RPC cluster with flash failover support and other such magic? Use the
standard library and write it yourself, Wedge would not be a good fit.


Usage:

.. code-block:: go

    func HelloWorld(req *http.Request) interface{} {
	    return "Hello world!"
    }

    func main() {

	wedge.Patterns(
		wedge.URL("/jsonhello", "HelloWorld", HelloWorld, wedge.JSON),
		wedge.URL("/", "HelloWorld", HelloWorld, wedge.HTTP),
	)

	wedge.Run("12345", 30)
    }
