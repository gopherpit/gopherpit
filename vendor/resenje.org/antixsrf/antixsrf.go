package antixsrf // import "resenje.org/antixsrf"

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"io"
	"net/http"
	"strings"
)

var (
	XSRFCookieName    = "secid"
	XSRFHeaderName    = "X-Secid"
	XSRFFormFieldName = "secid"

	safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

	ErrNoReferer      = errors.New("antixsrf: missing referer header")
	ErrInvalidReferer = errors.New("antixsrf: invalid referer header")
	ErrInvalidToken   = errors.New("antixsrf: invalid xsrf token")
	ErrMissingCookie  = errors.New("antixsrf: missing xsrf cookie")
	ErrMissingHeader  = errors.New("antixsrf: missing xsrf header")
)

func Verify(r *http.Request) error {
	if contains(safeMethods, r.Method) {
		return nil
	}

	referer, err := r.URL.Parse(r.Header.Get("Referer"))
	if err != nil {
		return err
	}
	if referer.String() == "" {
		return ErrNoReferer
	}

	if !(r.Host == referer.Host) {
		return ErrInvalidReferer
	}

	token, err := r.Cookie(XSRFCookieName)
	if err != nil {
		return err
	}

	if contains([]string{"application/x-www-form-urlencoded", "multipart/form-data"}, r.Header.Get("Content-Type")) {
		if r.FormValue(XSRFFormFieldName) == token.Value {
			return nil
		}
	}

	if token.Value == "" {
		return ErrMissingCookie
	}

	header := r.Header.Get(XSRFHeaderName)

	if header == "" {
		return ErrMissingHeader
	}

	if header == token.Value {
		return nil
	}

	return ErrInvalidToken
}

func Generate(w http.ResponseWriter, r *http.Request, path string) {
	if _, err := r.Cookie(XSRFCookieName); err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:   XSRFCookieName,
			Value:  newKey(),
			Path:   path,
			Secure: r.TLS != nil,
		})
	}
}

func newKey() string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString(GenerateRandomKey(16)), "=")
}

func GenerateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
