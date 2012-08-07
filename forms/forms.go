package forms

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

type FormMetadata struct {
	name   string
	action string
	method string
	submit bool
}

// NewFormMetadata encapsulates the data which needs to be passed to the Form
// constructor. We make it here in case many forms need to share Metadata, so
// it becomes possible to share vvalues of FormMetadata with multiple Forms.
func NewFormMetadata(name, action, method string, submit bool) FormMetadata {
	return FormMetadata{
		name:   name,
		action: action,
		method: method,
		submit: submit,
	}
}

type Field interface {
	Validate(interface{}, *http.Request) bool       // Tells us whether the form is valid
	Name() string                                   // Returns a name for the field
	Convert(interface{}, *http.Request) interface{} // Converts the form data into Go objects
	Display() string                                // Asks the field to display itself.
}

type Form struct {
	md         FormMetadata
	fields     map[string]Field
	fieldslice []Field
	req        *http.Request
}

// Fields allows you to iterate through the fields and have a custom order, or specialized
// output versus using the Display method.
func (f Form) Fields() []Field {
	return f.fieldslice
}

// Display iterates through all the Fields and calls their Display method
// adding their return values to a buffer and flushing that to the caller.
func (f Form) Display() string {
	buf := bytes.NewBufferString("")
	buf.WriteString(
		fmt.Sprintf(`<form name="%s" action="%s" method="%s">`,
			f.md.name, f.md.action, f.md.method,
		),
	)

	for _, field := range f.fieldslice {
		buf.WriteString(field.Display())
		buf.WriteString(`<br/>`)
	}
	if f.md.submit {
		buf.WriteString(`<input type="submit" value="Submit">`)
	}
	buf.WriteString(`</form>`)
	return buf.String()
}

// Validate takes the incoming request object and checks if the form
// included with it.
//
// Validate works on the Field interface. Considering that we will have
// quite a lot of field types, which need to be grouped onto a Form.
func (f Form) Validate(req *http.Request) bool {
	req.ParseForm()

	inputForm := req.Form
	for key, value := range f.fields {
		if _, ok := inputForm[key]; !ok {
			log.Println("Key not in inputForm:", key)
			return false
		}
		if !value.Validate(inputForm[key], req) {
			log.Println("Failed to validate:", key)
			return false
		}
	}

	f.req = req
	return true
}

// Form iterates through all the Fields on the Form and calls their
// Convert method and assigns the result in a map.
func (f Form) Convert(req *http.Request) map[string]interface{} {
	inputForm := req.Form
	outform := make(map[string]interface{})
	for key, value := range f.fields {
		outform[key] = value.Convert(inputForm[key], req)
	}
	return outform
}

// NewForm creates an instance of a *Form and returns a pointer to it.
func NewForm(md FormMetadata, forms ...Field) *Form {
	newForm := Form{
		md:         md,
		fields:     make(map[string]Field),
		fieldslice: forms,
	}
	for _, f := range forms {
		newForm.fields[f.Name()] = f
	}

	return &newForm
}

type Text struct {
	name      string
	long_name string
	max_len   int
}

func TextField(name, long_name string, l int) Field {
	return Text{name, long_name, l}
}

func (t Text) Validate(key interface{}, f *http.Request) bool {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error validating Text value")
		return false
	}
	if len(k[0]) < t.max_len {
		return true
	}
	log.Println("TextField didn't validate")
	return false
}

func (t Text) Convert(key interface{}, f *http.Request) interface{} {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error converting Text value")
		return false
	}
	return k[0]
}

func (t Text) Name() string {
	return t.name
}

func (t Text) Display() string {
	return fmt.Sprintf(`%s: <input type="text" name="%s" />`, t.long_name, t.name)
}

type Radio struct {
	name          string
	choices       map[string]string
	choices_slice []choice_options
}

// RadioField creates a Radio value which will have it's fields properly initialized
// with the choices which are passed to it.
func RadioField(name string, choices ...choice_options) Field {
	m := initMultipleOptions(choices)
	return Radio{name, m, choices}
}

func (r Radio) Validate(key interface{}, req *http.Request) bool {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error validating Radio value")
		return false
	}
	if _, ok := r.choices[k[0]]; ok {
		return true
	}
	log.Println("Error validating Radio value: Entry not in map.")
	return false
}

func (r Radio) Convert(key interface{}, req *http.Request) interface{} {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error converting Radio value")
		return false
	}
	return k[0]
}

func (r Radio) Name() string {
	return r.name
}

func (r Radio) Display() string {
	return writeMultipleOptions(r, r.choices_slice, "radio")
}

type Check struct {
	name          string
	min_len       int
	choices       map[string]string
	choices_slice []choice_options
}

type choice_options struct {
	choice  string
	name    string
	checked string
}

func Choice(choice, name string, checked bool) choice_options {
	checkstr := ""
	if checked {
		checkstr = `checked="checked"`
	}

	return choice_options{choice, name, checkstr}
}

// CheckField creates a Check value which will have it's fields properly initialized
// with the choices which are passed to it.
func CheckField(name string, min int, choices ...choice_options) Field {
	m := initMultipleOptions(choices)
	return Check{name, min, m, choices}
}

func (c Check) Validate(key interface{}, req *http.Request) bool {

	k, ok := key.([]string)
	if !ok {
		log.Println("CheckField didn't validate: Assert")
		return false
	}

	if len(k) < c.min_len {
		log.Println("CheckField didn't validate: Length")
		return false
	}

	for _, value := range k {
		if _, ok := c.choices[value]; !ok {
			log.Println("CheckField didn't validate: Value not in map.")
			return false
		}
	}

	return true
}

func (c Check) Convert(key interface{}, req *http.Request) interface{} {
	k, ok := key.([]string)
	if !ok {
		log.Printf("Error converting a Check value")
		return false
	}
	return k
}

func (c Check) Name() string {
	return c.name
}

func (c Check) Display() string {
	return writeMultipleOptions(c, c.choices_slice, "checkbox")
}

type Password struct {
	name      string
	long_name string
	min       int
	max       int
}

func PasswordField(name, long_name string, min, max int) Password {
	return Password{
		name:      name,
		long_name: long_name,
		min:       min,
		max:       max,
	}
}

func (p Password) Validate(key interface{}, req *http.Request) bool {
	val, ok := key.([]string)
	if !ok {
		log.Println("Error validating Password value")
		return false
	}
	if (len(val[0]) >= p.min) && (len(val[0]) <= p.max) {
		return true
	}
	log.Println("Failure to validate Password: Length")
	return false
}

func (p Password) Convert(key interface{}, req *http.Request) interface{} {
	val, ok := key.([]string)
	if !ok {
		log.Println("Error converting Password value")
		return false
	}
	return val[0]
}

func (p Password) Name() string {
	return p.name
}

func (p Password) Display() string {
	return fmt.Sprintf(`%s: <input type="password" name="%s" />`, p.long_name, p.name)
}

type Combo struct {
	name          string
	long_name     string
	choices       map[string]string
	choices_slice []choice_options
}

func ComboField(name, long_name string, choices ...choice_options) Field {
	m := initMultipleOptions(choices)
	return Combo{name, long_name, m, choices}
}

func (c Combo) Validate(key interface{}, req *http.Request) bool {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error validating Combo: assert")
	}
	if _, ok := c.choices[k[0]]; ok {
		return true
	}
	return false
}

func (c Combo) Convert(key interface{}, req *http.Request) interface{} {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error converting Combo: assert")
	}
	return k[0]
}

func (c Combo) Name() string {
	return c.name
}

func (c Combo) Display() string {
	buf := bytes.NewBufferString("")
	buf.WriteString(
		fmt.Sprintf(`%s: <select name="%s">`, c.long_name, c.name),
	)
	for _, choice := range c.choices_slice {
		buf.WriteString(
			fmt.Sprintf(`<option value="%s">%s</option>`,
				choice.name, choice.choice,
			),
		)
	}
	buf.WriteString(`</select>`)
	return buf.String()
}

// writeMultipleOptions is a helper method which is used for Fields which have
// a very similar internal datastructure and a very similar output format.
//
// It's useful for things which vary very little in their HTML representation.
func writeMultipleOptions(object Field, choices []choice_options, ftype string) string {
	buf := bytes.NewBufferString("")
	for _, choice := range choices {
		buf.WriteString(
			fmt.Sprintf(`%s: <input type="%s" name="%s" value="%s" %s /><br />`,
			choice.choice, ftype, object.Name(), choice.name, choice.checked,
			),
		)
	}
	return buf.String()
}

// initMultipleOptions is a helper method which is used for Fields which have
// a very similar internal datastructure so they can be initilized in the same
// way.
func initMultipleOptions(choices []choice_options) (map[string]string) {
	m := make(map[string]string)
	for _, choice := range choices {
		m[choice.name] = choice.choice
	}
	return m
}
