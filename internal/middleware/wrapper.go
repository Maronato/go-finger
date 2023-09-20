package middleware

import (
	"fmt"
	"net/http"
)

type ResponseWrapper struct {
	http.ResponseWriter

	status int
}

func WrapResponseWriter(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{w, 0}
}

func (w *ResponseWrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWrapper) Status() int {
	return w.status
}

func (w *ResponseWrapper) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	size, err := w.ResponseWriter.Write(b)
	if err != nil {
		return 0, fmt.Errorf("error writing response: %w", err)
	}

	return size, nil
}

func (w *ResponseWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
