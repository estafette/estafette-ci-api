package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	prometheusAddress     = flag.String("metrics-listen-address", ":9001", "The address to listen on for Prometheus metrics requests.")
	prometheusMetricsPath = flag.String("metrics-path", "/metrics", "The path to listen for Prometheus metrics requests.")
	apiAddress            = flag.String("api-listen-address", ":5000", "The address to listen on for api HTTP requests.")

	// seed random number
	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	// define prometheus counter
	httpRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of handled http requests.",
		},
		[]string{"code", "handler"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(httpRequestTotal)
}

func main() {

	fmt.Printf("Starting estafette-ci-api (version=%v, branch=%v, revision=%v, buildDate=%v, goVersion=%v)\n", version, branch, revision, buildDate, goVersion)

	// start prometheus
	go func() {
		fmt.Printf("Serving Prometheus metrics at %v%v...\n", *prometheusAddress, *prometheusMetricsPath)
		http.Handle(*prometheusMetricsPath, promhttp.Handler())
		log.Fatal(http.ListenAndServe(*prometheusAddress, nil))
	}()

	fmt.Printf("Listening at %v for api calls...\n", *apiAddress)
	http.HandleFunc("/webhook/github", githubWebhookHandler)
	http.HandleFunc("/liveness", livenessHandler)
	http.HandleFunc("/readiness", readinessHandler)
	log.Fatal(http.ListenAndServe(*apiAddress, nil))
}

func githubWebhookHandler(w http.ResponseWriter, r *http.Request) {

	// https://developer.github.com/webhooks/

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}

	fmt.Println("Received webhook from GitHub...")
	fmt.Printf("url: %v\n", r.URL)
	fmt.Printf("header: %v\n", r.Header)
	fmt.Printf("body: %v\n", string(body))

	fmt.Fprintf(w, "Zero distortion!")
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm alive!")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "I'm ready!")
}
