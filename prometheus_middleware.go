// Looted from: https://raw.githubusercontent.com/albertogviana/prometheus-middleware/master/prometheus.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dflBuckets = []float64{100, 300, 1000, 3000}
)

const (
	requestName = "http_requests_total"
	latencyName = "http_request_duration_milliseconds"
)

// PromMiddlewareOpts specifies options how to create new PrometheusMiddleware.
type PromMiddlewareOpts struct {
	// Buckets specifies an custom buckets to be used in request histograpm.
	Buckets []float64
}

// PrometheusMiddleware specifies the metrics that is going to be generated
type PrometheusMiddleware struct {
	request *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware instance
func NewPrometheusMiddleware(opts PromMiddlewareOpts) *PrometheusMiddleware {
	var prometheusMiddleware PrometheusMiddleware

	prometheusMiddleware.request = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: requestName,
			Help: "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
		},
		[]string{"code", "method", "path"},
	)

	if err := prometheus.Register(prometheusMiddleware.request); err != nil {
		log.Println("prometheusMiddleware.request was not registered:", err)
	}

	buckets := opts.Buckets
	if len(buckets) == 0 {
		buckets = dflBuckets
	}

	prometheusMiddleware.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    latencyName,
		Help:    "How long it took to process the request, partitioned by status code, method and HTTP path.",
		Buckets: buckets,
	},
		[]string{"code", "method", "path"},
	)

	if err := prometheus.Register(prometheusMiddleware.latency); err != nil {
		log.Println("prometheusMiddleware.latency was not registered:", err)
	}

	return &prometheusMiddleware
}

// InstrumentHandlerDuration is a middleware that wraps the http.Handler and it record
// how long the handler took to run, which path was called, and the status code.
// This method is going to be used with gorilla/mux.
func (p *PrometheusMiddleware) InstrumentHandlerDuration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sleepMsString := mux.Vars(r)["sleepMs"]
		ms, err := time.ParseDuration(fmt.Sprintf("%sms", sleepMsString))

		if err != nil {
			withRealObserver(p, next, w, r)
		} else {
			withFakeObserver(p, next, w, r, ms)
		}
	})
}

func withFakeObserver(p *PrometheusMiddleware, next http.Handler, w http.ResponseWriter, r *http.Request, ms time.Duration) {
	fakeTime := float64(ms / time.Millisecond)
	delegate := &responseWriterDelegator{ResponseWriter: w}
	rw := delegate

	next.ServeHTTP(rw, r)

	route := mux.CurrentRoute(r)
	path, _ := route.GetPathTemplate()
	code := sanitizeCode(delegate.status)
	method := sanitizeMethod(r.Method)

	go p.request.WithLabelValues(
		code,
		method,
		path,
	).Inc()

	go p.latency.WithLabelValues(
		code,
		method,
		path,
	).Observe(fakeTime)
}

func withRealObserver(p *PrometheusMiddleware, next http.Handler, w http.ResponseWriter, r *http.Request) {
	begin := time.Now()
	delegate := &responseWriterDelegator{ResponseWriter: w}
	rw := delegate

	next.ServeHTTP(rw, r)

	route := mux.CurrentRoute(r)
	path, _ := route.GetPathTemplate()
	code := sanitizeCode(delegate.status)
	method := sanitizeMethod(r.Method)

	go p.request.WithLabelValues(
		code,
		method,
		path,
	).Inc()

	go p.latency.WithLabelValues(
		code,
		method,
		path,
	).Observe(float64(time.Since(begin)) / float64(time.Millisecond))
}

type responseWriterDelegator struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func (r *responseWriterDelegator) WriteHeader(code int) {
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseWriterDelegator) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

func sanitizeMethod(m string) string {
	return strings.ToLower(m)
}

func sanitizeCode(s int) string {
	return strconv.Itoa(s)
}
