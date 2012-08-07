Package forms is an extension to wedge which allows the easy processing, conversion
and creation of HTML web forms.

Currently there are very few types, however, adding new types is as simple as satisfying
the Field interface.

The Field interface as as follows:

  .. code-block:: go 

      type Field interface {
      	  Validate(interface{}, *http.Request) bool
	      Name() string
          Convert(interface{}, *http.Request) interface{}
          Display() string
      }

A types Validate method must return a boolean value indicating whether it's associated
Field is either a valid representation of that field, or if the value that the form
holds is valid. If Validation fails on any field, the entire validation fails for the
form.

A types Name simply returns the .name field associated with it.

A types Convert method uses the form again to change from the textual representation
of itself in the form to the Go object form. This returns an interface so you will
need to assert the type on the returned value. It's possible that I will eventually
have it so that upon creating Forms with NewForm we could use reflect to create a new
type and have the Convert method fill in the fields of that type. This is a long way
off and I'm not sure if it's entirely possible but we'll see.

A types Display returns a string of how the Field should be represented in HTML.

As you can see, satisfying this interface is quite simple.