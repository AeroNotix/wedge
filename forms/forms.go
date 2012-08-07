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

	inputForm := req.Form
	for key, value := range f.fields {
		if _, ok := inputForm[key]; !ok {
			return false
		}
		if !value.Validate(inputForm[key], req) {
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

func NewForm(name string, md FormMetadata, forms ...Field) *Form {
	newForm := Form{
		fields:     make(map[string]Field),
		fieldslice: []Field{},
	}
	for _, f := range forms {
		newForm.fieldslice = append(newForm.fieldslice, f)
		newForm.fields[f.Name()] = f
	}

	return &newForm
}

type Text struct {
	name    string
	max_len int
}

func TextField(n string, l int) Field {
	return Text{n, l}
}

func (t Text) Validate(key interface{}, f *http.Request) bool {
	k, ok := key.([]string)
	if !ok {
		log.Println("Error validating Text value")
		return false
	}

	if len(f.FormValue(k[0])) <= t.max_len {
		return true
	}
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
	return fmt.Sprintf(`<input type="text" name="%s" />`, t.name)
}

type Radio struct {
	name    string
	choices map[string]bool
}

func RadioField(name string, choices []string) Field {
	m := make(map[string]bool)
	for _, choice := range choices {
		m[choice] = true
	}
	return Radio{name, m}
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
	return false
}

func (r Radio) Convert(key interface{}, req *http.Request) interface{} {
	k, ok := key.(string)
	if !ok {
		log.Println("Error converting Radio value")
	}
	return k
}

func (r Radio) Name() string {
	return r.name
}

func (r Radio) Display() string {
	buf := bytes.NewBufferString("")
	for choice, _ := range r.choices {
		buf.WriteString(
			fmt.Sprintf(`<input type="radio" name="%s" value="%s" />`,
				r.name, choice,
			),
		)
	}
	return buf.String()
}

type Check struct {
	name    string
	min_len int
	choices map[string]bool
}

func CheckField(name string, choices []string, min int) Field {
	m := make(map[string]bool)
	for _, choice := range choices {
		m[choice] = true
	}
	return Check{name, min, m}
}

func (c Check) Validate(key interface{}, req *http.Request) bool {

	k, ok := key.([]string)
	if !ok {
		return false
	}

	if len(k) < c.min_len {
		return false
	}

	for _, value := range k {
		if _, ok := c.choices[value]; !ok {
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
	buf := bytes.NewBufferString("")
	for choice, _ := range c.choices {
		buf.WriteString(
			fmt.Sprintf(`<input type="checkbox" name="%s" value="%s" />`,
				c.name, choice,
			),
		)
	}
	return buf.String()
}