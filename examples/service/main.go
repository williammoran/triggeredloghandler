package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"syscall"

	"github.com/google/uuid"
	"github.com/williammoran/triggeredloghandler"
)

func main() {
	// set PPROF to a base filename in the environment to
	// trigger a CPU profile at program exit
	profileFile := os.Getenv("PPROF")
	if profileFile != "" {
		f0, err := os.Create(profileFile + ".pprof")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f0.Close()
		if err := pprof.StartCPUProfile(f0); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	// If TRIGGER_LEVEL is unset, then the triggered logger
	// is not used. If it is set, it is used as the level
	// at which to trigger
	levelStr := os.Getenv("TRIGGER_LEVEL")
	useTriggered := false
	var level slog.Level
	if levelStr != "" {
		err := level.UnmarshalText([]byte(levelStr))
		if err != nil {
			log.Panicf("Invalid TRIGGER_LEVEL='%s'", levelStr)
		}
		useTriggered = true
	}
	h := service{
		useTriggered:   useTriggered,
		triggeredLevel: level,
		handler:        slog.NewTextHandler(os.Stdout, nil),
	}
	http.HandleFunc("GET /sqrt/{num}", h.HandleSqrt)
	// This verbose process of starting the HTTP server
	// ensures clean shutdown, which is necessary to allow
	// the pprof file to be written before exit, but is also
	// a good idea in general.
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	server := http.Server{Addr: ":8000"}
	go server.ListenAndServe()
	<-signalChannel
	server.Shutdown(context.Background())
}

type service struct {
	useTriggered   bool
	triggeredLevel slog.Level
	handler        slog.Handler
}

// HandleSqrt is an example HTTP handler to showcase typical
// use of the Triggered handler
func (h *service) HandleSqrt(w http.ResponseWriter, r *http.Request) {
	var handler slog.Handler
	if h.useTriggered {
		// If configured to use the triggered logger, then
		// create a new one. It's critical that a new
		// triggered logger is created for each request,
		// otherwise memory for unlogged messages will never
		// be reclaimed
		handler = triggeredloghandler.NewTriggeredLogHandler(
			h.handler,
			uuid.New().String(),
			h.triggeredLevel,
		)
	} else {
		// If triggered logging is not enabled, just use a
		// standard handler, but add a stream key for
		// consistency
		handler = h.handler.WithAttrs([]slog.Attr{{
			Key:   triggeredloghandler.TriggeredLogStreamIDKey,
			Value: slog.StringValue(uuid.New().String()),
		}})
	}
	logger := slog.New(handler)
	// Logging is intentionally excessive to demonstrate functionality
	logger.Debug("Starting HandleSqrt")
	// The actual work of this handler is to read a floating point
	// number from the request path and return its square root
	io.ReadAll(r.Body)
	numStr := r.PathValue("num")
	logger.Info("Sqrt value", "value", numStr)
	// Method 1 of causing the handler to error, provide a
	// value not parseable as a float
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		// Note that it's not necessary to include the actual value
		// here, since the previous Info() message will include it
		// and will be grouped by stream id.
		logger.Error("failed to parse value", "error", err.Error())
		http.Error(w, "Bad value", http.StatusBadRequest)
		return
	}
	// Method 2 of causing the handler to error, provide a
	// negative number
	if num < 0 {
		logger.Info("can not sqrt negative value")
		http.Error(w, "Bad value", http.StatusBadRequest)
		return
	}
	result := fmt.Sprintf("%f", math.Sqrt(num))
	// More overly verbose logging
	logger.Debug("Result", "value", result)
	_, err = w.Write([]byte(result))
	if err != nil {
		logger.Error("failed to return result", "error", err.Error())
		return
	}
}
