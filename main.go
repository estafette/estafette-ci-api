package main

import (
	"flag"
	"fmt"
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
	addr = flag.String("listen-address", ":9101", "The address to listen on for HTTP requests.")

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
		fmt.Println("Serving Prometheus metrics at :9101/metrics...")
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(*addr, nil))
	}()
}
