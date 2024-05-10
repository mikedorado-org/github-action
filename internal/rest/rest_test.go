package rest

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestHello(t *testing.T) {
	name := "Gladys"
	r, err := http.NewRequest("GET", "hello/Gladys", nil)
	w := httptest.NewRecorder()
	want := regexp.MustCompile("Hello, " + name + ".*")

	getHello(w, r)

	body := w.Body.String()
	if !want.MatchString(body) || err != nil {
		t.Fatalf(`Hello("Gladys") = %q, %v, want match for %#q, nil`, body, err, want)
	}
}

func TestNotFound(t *testing.T) {
	r, err := http.NewRequest("GET", "notarealendpoint", nil)
	w := httptest.NewRecorder()

	notFound(w, r)

	if w.Result().StatusCode != http.StatusNotFound || err != nil {
		t.Fatalf(`Not found should have returns 404, got %d, %v`, w.Result().StatusCode, err)
	}
}
