package forms

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/asaskevich/govalidator"
)

// Form creates a custom form struct
type Form struct {
	url.Values
	Errors errors
}

// Valid returns true if there are no errors
func (f *Form) Valid() bool {
	return len(f.Errors) == 0
}
func New(data url.Values) *Form {
	return &Form{
		data,
		errors(map[string][]string{}),
	}
}

func (f *Form) Required(fields ...string) {
	for _, field := range fields {
		value := f.Get(field)
		if strings.TrimSpace(value) == "" {
			f.Errors.Add(field, "This field cannot be blank")
		}
	}
}

// Has checks if form field is in post and not empty
func (f *Form) Has(field string, r *http.Request) bool {
	x := f.Get(field)
	if x == "" {
		f.Errors.Add(field, "This field cannot be blank")
		return false
	}
	return true
}

// MinLength checks if given field has admissible length
func (f *Form) MinLength(field string, length int, r *http.Request) bool {
	x := f.Get(field)
	if len(x) < length {
		f.Errors.Add(field, fmt.Sprintf("This field must be at least %d characters long", length))
		return false
	}
	return true
}

// IsEmail checks if given field is an email
func (f *Form) IsEmail(field string) bool {
	if !govalidator.IsEmail(f.Get(field)) {
		f.Errors.Add(field, "Invalid email address")
		return false
	}
	return true
}

// IsNum checks if given field is a number
func (f *Form) IsNum(field string) bool {
	char := []rune(f.Get(field))
	if !unicode.IsNumber(char[0]) {
		f.Errors.Add(field, "This field must be a number")
		return false
	}
	return true
}
