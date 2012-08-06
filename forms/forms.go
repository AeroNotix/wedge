package forms

import (
	"log"
	"net/http"
)

type Field interface {
	Validate(interface{}, *http.Request) bool
	Name() string
	Convert(interface{}, *http.Request) interface{}
}

type Form struct {
	fields map[string]Field
	req    *http.Request
}

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

func (f Form) Convert(req *http.Request) map[string]interface{} {
	inputForm := req.Form
	outform := make(map[string]interface{})
	for key, value := range f.fields {
		outform[key] = value.Convert(inputForm[key], req)
	}
	return outform
}

func NewForm(forms ...Field) *Form {
	newForm := Form{
		fields: make(map[string]Field),
	}
	for _, f := range forms {
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
