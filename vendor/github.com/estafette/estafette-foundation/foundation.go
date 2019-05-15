package foundation

import (
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	goVersion = runtime.Version()

	// seed random number
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// InitLogging initializes logging to log everything as json
func InitLogging(app, version, branch, revision, buildDate string) {
	// log as severity for stackdriver logging to recognize the level
	zerolog.LevelFieldName = "severity"

	// set some default fields added to all logs
	log.Logger = zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", app).
		Str("version", version).
		Logger()

	// use zerolog for any logs sent via standard log library
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	// log startup message
	log.Info().
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msgf("Starting %v...", app)
}

// InitMetrics initializes the prometheus endpoint /metrics on port 9101
func InitMetrics() {
	// start prometheus
	go func() {
		log.Debug().
			Str("port", ":9101").
			Msg("Serving Prometheus metrics...")

		http.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(":9101", nil); err != nil {
			log.Fatal().Err(err).Msg("Starting Prometheus listener failed")
		}
	}()
}

// InitGracefulShutdownHandling generates the channel that listens to SIGTERM and a waitgroup to use for finishing work when shutting down
func InitGracefulShutdownHandling() (gracefulShutdown chan os.Signal, waitGroup *sync.WaitGroup) {

	// define channel used to gracefully shutdown the application
	gracefulShutdown = make(chan os.Signal)

	signal.Notify(gracefulShutdown, syscall.SIGTERM, syscall.SIGINT)

	waitGroup = &sync.WaitGroup{}

	return gracefulShutdown, waitGroup
}

// HandleGracefulShutdown waits for SIGTERM to unblock gracefulShutdown and waits for the waitgroup to await pending work
func HandleGracefulShutdown(gracefulShutdown chan os.Signal, waitGroup *sync.WaitGroup) {

	signalReceived := <-gracefulShutdown
	log.Info().
		Msgf("Received signal %v. Waiting for running tasks to finish...", signalReceived)

	waitGroup.Wait()

	log.Info().Msg("Shutting down...")
}

// ApplyJitter adds +-25% jitter to the input
func ApplyJitter(input int) (output int) {

	deviation := int(0.25 * float64(input))

	return input - deviation + r.Intn(2*deviation)
}

// WatchForFileChanges waits for a change to the provided file path and then executes the function
func WatchForFileChanges(filePath string, functionOnChange func(fsnotify.Event)) {
	// copied from https://github.com/spf13/viper/blob/v1.3.1/viper.go#L282-L348
	initWG := sync.WaitGroup{}
	initWG.Add(1)
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal().Err(err).Msg("Creating file system watcher failed")
		}
		defer watcher.Close()

		// we have to watch the entire directory to pick up renames/atomic saves in a cross-platform way
		file := filepath.Clean(filePath)
		fileDir, _ := filepath.Split(file)
		realFile, _ := filepath.EvalSymlinks(filePath)

		eventsWG := sync.WaitGroup{}
		eventsWG.Add(1)
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok { // 'Events' channel is closed
						eventsWG.Done()
						return
					}
					currentFile, _ := filepath.EvalSymlinks(filePath)
					// we only care about the key file with the following cases:
					// 1 - if the key file was modified or created
					// 2 - if the real path to the key file changed (eg: k8s ConfigMap/Secret replacement)
					const writeOrCreateMask = fsnotify.Write | fsnotify.Create
					if (filepath.Clean(event.Name) == file &&
						event.Op&writeOrCreateMask != 0) ||
						(currentFile != "" && currentFile != realFile) {
						realFile = currentFile

						functionOnChange(event)
					} else if filepath.Clean(event.Name) == file &&
						event.Op&fsnotify.Remove&fsnotify.Remove != 0 {
						eventsWG.Done()
						return
					}

				case err, ok := <-watcher.Errors:
					if ok { // 'Errors' channel is not closed
						log.Printf("watcher error: %v\n", err)
					}
					eventsWG.Done()
					return
				}
			}
		}()
		watcher.Add(fileDir)
		initWG.Done()   // done initalizing the watch in this go routine, so the parent routine can move on...
		eventsWG.Wait() // now, wait for event loop to end in this go-routine...
	}()
	initWG.Wait() // make sure that the go routine above fully ended before returning
}
