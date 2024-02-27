package main

import (
	"log/slog"
	"os"

	"github.com/williammoran/triggeredloghandler"
)

// This file serves as a basic demonstration of the functionality
// of the TriggeredLogHandler

func main() {
	// For simplicity's sake, messages will be logged using go's
	// standard text handler.
	// Note that this could just as easily be any handler, such
	// as the JSON handler or some vendor-specific log aggregator.
	targetLogger := slog.NewTextHandler(os.Stdout, nil)
	trigger := triggeredloghandler.NewTriggeredLogHandler(targetLogger, "1", slog.LevelError)
	logger := slog.New(trigger)
	logger.Debug("Debug message 1")
	// At this point in the code, the Debug message has not yet
	// been logged because the severity is not high enough to
	// trigger
	logger.Error("Error message")
	// After this Error(), both the Debug and the Error will be
	// logged in the order they were reported.

	// Creating a new TriggereLogHandler resets the trigger
	trigger = triggeredloghandler.NewTriggeredLogHandler(targetLogger, "2", slog.LevelError)
	logger = slog.New(trigger)
	logger.Debug("Debug message 2")
	// Note this debug message will never be logged because no
	// messages were ever severe enough to trigger
	// When the function returns, the garbage collector will clear
	// all the messages in the TriggeredLogHandler
}
