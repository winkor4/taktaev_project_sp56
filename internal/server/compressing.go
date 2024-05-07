package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipW *gzip.Writer
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	return &gzipResponseWriter{
		ResponseWriter: w,
		gzipW:          gzip.NewWriter(w),
	}
}

func (gw *gzipResponseWriter) Header() http.Header {
	return gw.ResponseWriter.Header()
}

func (gw *gzipResponseWriter) Write(p []byte) (int, error) {
	return gw.gzipW.Write(p)
}

func (gw *gzipResponseWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		gw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}
	gw.ResponseWriter.WriteHeader(statusCode)
}

func (gw *gzipResponseWriter) Close() error {
	return gw.gzipW.Close()
}

type gzipReader struct {
	ioR   io.ReadCloser
	gzipR *gzip.Reader
}

func newGzipReader(ioR io.ReadCloser) (*gzipReader, error) {
	gzipR, err := gzip.NewReader(ioR)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		ioR:   ioR,
		gzipR: gzipR,
	}, nil
}

func (gzipR *gzipReader) Read(p []byte) (n int, err error) {
	return gzipR.gzipR.Read(p)
}

func (gzipR *gzipReader) Close() error {
	if err := gzipR.ioR.Close(); err != nil {
		return err
	}
	return gzipR.gzipR.Close()
}

func gzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := w

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gw := newGzipResponseWriter(w)
			rw = gw
			defer gw.Close()
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzipR, err := newGzipReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = gzipR
			defer gzipR.Close()
		}

		h.ServeHTTP(rw, r)
	})
}
