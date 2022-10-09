package main

import (
	"fmt"
	"net/http"
	"testing"
)

var myH myHandler

func TestNoSurf(t *testing.T) {
	handler := NoSurf(&myH)

	switch v := handler.(type) {
	case http.Handler:
		// Fine, do nothing
	default:
		t.Error(fmt.Sprintf("type is not http.Handler, but is type %s", v))
	}
}

func TestSessionLoad(t *testing.T) {
	handler := SessionLoad(&myH)

	switch v := handler.(type) {
	case http.Handler:
	// Fine, do nothing
	default:
		t.Error(fmt.Sprintf("type is not http.Handler, but is type %s", v))
	}
}
