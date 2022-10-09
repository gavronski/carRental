package forms

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestForm_Valid(t *testing.T) {
	r := httptest.NewRequest("POST", "/test", nil)
	form := New(r.PostForm)

	if !form.Valid() {
		t.Error("form should be valid")
	}
}

func TestForm_Required(t *testing.T) {
	r := httptest.NewRequest("POST", "/test", nil)
	form := New(r.PostForm)
	form.Required("test1", "test2", "test3")

	if form.Valid() {
		t.Error("form should not be valid")
	}

	values := url.Values{}
	values.Add("test1", "test1")
	values.Add("test2", "test2")
	values.Add("test3", "test3")

	r = httptest.NewRequest("POST", "/test", nil)
	form = New(values)
	form.Required("test1", "test2", "test3")

	if !form.Valid() {
		t.Error("form should be valid")
	}

}

func TestForm_Has(t *testing.T) {
	r := httptest.NewRequest("POST", "/test", nil)
	form := New(r.PostForm)

	hasField := form.Has("test1", r)
	if hasField {
		t.Error("Fomr should not have this field")
	}

	values := url.Values{}
	values.Add("test1", "test1")
	form = New(values)

	if !form.Has("test1", r) {
		t.Error("Form shoud have this field")
	}
}

func TestForm_MinLength(t *testing.T) {
	r := httptest.NewRequest("POST", "/test", nil)
	values := url.Values{}
	values.Add("test1", "abc")
	values.Add("test2", "zcbm")

	form := New(values)

	minLength := form.MinLength("test1", 9, r)
	if minLength {
		t.Error("Field does not have min length")
	}

	minLength = form.MinLength("test2", 3, r)
	if !minLength {
		t.Error("Field have at least 3 chars")
	}
}

func TestForm_IsEmail(t *testing.T) {
	values := url.Values{}
	values.Add("email1", "xyz")
	values.Add("email2", "xyz@gmail.com")

	form := New(values)

	if form.IsEmail("email1") {
		t.Error("This field is not an email")
	}
	if !form.IsEmail("email2") {
		t.Error("This field should be consider as an email")
	}
}

func TestForm_Get(t *testing.T) {
	values := url.Values{}
	values.Add("email1", "xyz")
	form := New(values)

	form.IsEmail("email1")

	err := form.Errors.Get("email1")
	if err != "Invalid email address" {
		t.Error("First error should be - 'Invalid email address'")
	}

	err = form.Errors.Get("email2")
	if err != "" {
		t.Error("First error should be empty", err)
	}
}

func Test_IsNum(t *testing.T) {
	values := url.Values{}
	values.Add("x", "test")
	values.Add("y", "111")

	form := New(values)

	if form.IsNum("x") {
		t.Error("Should be worng but it's fine")
	}

	if !form.IsNum("y") {
		t.Error("Is number but got false")
	}
}
