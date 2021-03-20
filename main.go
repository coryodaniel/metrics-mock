package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type StatusResponse struct {
	CodeText string
	Code     int
	Method   string
}

type LatencyResponse struct {
	Duration time.Duration
	Method   string
}

var (
	//go:embed html/index.html
	indexPageEmbed string
	//go:embed html/status.html.tmpl
	statusTemplateEmbed string
	//go:embed html/latency.html.tmpl
	latencyTemplateEmbed string
	//go:embed VERSION
	version         string
	addr            = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	indexPage       = []byte(indexPageEmbed)
	statusTemplate  = template.Must(template.New("status").Parse(statusTemplateEmbed))
	latencyTemplate = template.Must(template.New("latency").Parse(latencyTemplateEmbed))
)

func main() {
	flag.Parse()

	prometheusMiddleware := NewPrometheusMiddleware(PromMiddlewareOpts{})
	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(prometheusMiddleware.InstrumentHandlerDuration)
	router.Path("/").HandlerFunc(IndexHandler)
	router.Path("/metrics").Handler(promhttp.Handler())
	router.Path("/status/{statusCode}").HandlerFunc(StatusHandler)
	router.Path("/latency/{sleepMs}").HandlerFunc(LatencyHandler)

	fmt.Println(fmt.Sprintf("metrics-mock %s listening on %s ...", version, *addr))
	http.ListenAndServe(*addr, router)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(indexPage)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	statusCodeString := mux.Vars(r)["statusCode"]
	statusCode, err := strconv.Atoi(statusCodeString)
	if err != nil {
		statusCode = 400
	}

	statusText := http.StatusText(statusCode)
	assigns := StatusResponse{
		CodeText: statusText,
		Code:     statusCode,
		Method:   r.Method,
	}

	w.WriteHeader(statusCode)
	statusTemplate.Execute(w, assigns)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(fmt.Sprintf("%s %s", r.Method, r.RequestURI))
		next.ServeHTTP(w, r)
	})
}

func LatencyHandler(w http.ResponseWriter, r *http.Request) {
	sleepMsString := mux.Vars(r)["sleepMs"]
	ms, err := time.ParseDuration(fmt.Sprintf("%sms", sleepMsString))

	if err != nil {
		ms = 0
	}

	assigns := LatencyResponse{
		Duration: ms,
		Method:   r.Method,
	}

	w.WriteHeader(http.StatusOK)
	latencyTemplate.Execute(w, assigns)
}
