package triggeredloghandler

import (
	"context"
	"log/slog"
	"sync"
)

// TriggeredLogStreamIDKey is the name of the attribute used to
// identify logs that are part of a group of triggere logs
const TriggeredLogStreamIDKey = "log_stream_id"

// TriggeredLogHandler is a slog.Handler that tracks message streams
// and only logs them if a message is logged that exceeds the trigger
// threshold. It does not process messages directly, but delegates
// processing to a target handler. TriggeredLogHandler is only
// responsible for tracking messages and determining when logging
// is triggered.
type TriggeredLogHandler struct {
	target slog.Handler
	base   *baseTriggeredLogHandler
}

// baseTriggeredLogHandler is data common to all TriggeredLogHandlers
// in a tree
type baseTriggeredLogHandler struct {
	triggerLevel slog.Level
	triggered    bool
	backlog      []backlogRecord
	mu           sync.Mutex
}

// backlogRecord is all the data needed to log a log record
type backlogRecord struct {
	ctx    context.Context
	target slog.Handler
	record slog.Record
}

// NewTriggeredLogHandler returns a TriggeredLogHandler at the root
// of a TriggeredLogHandler tree. It is initialized as not triggered.
// target is a handler where messages will be sent once triggered.
// The streamID is added as a value to the target handler to provide
// a consistent value across all messages using TriggeredLogStreamIDKey
// as the attribute key.
func NewTriggeredLogHandler(
	target slog.Handler,
	streamID string,
	triggerLevel slog.Level,
) *TriggeredLogHandler {
	target = target.WithAttrs([]slog.Attr{{
		Key:   TriggeredLogStreamIDKey,
		Value: slog.StringValue(streamID),
	}})
	return &TriggeredLogHandler{
		target: target,
		base: &baseTriggeredLogHandler{
			triggerLevel: triggerLevel,
		},
	}
}

// Enabled always returns true because the nature of the
// handler is that it processes all messages and if we
// ever returned false, slog won't send a message at all
func (tlh *TriggeredLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle stores the record if this handler hasn't been triggered
// yet, or sends any backlog including this record if it's been
// triggered
func (tlh *TriggeredLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level >= tlh.base.triggerLevel {
		tlh.base.triggered = true
	}
	tlh.base.mu.Lock()
	defer tlh.base.mu.Unlock()
	if tlh.base.triggered {
		if len(tlh.base.backlog) > 0 {
			err := tlh.forwardBacklog(ctx)
			if err != nil {
				tlh.base.backlog = append(
					tlh.base.backlog,
					backlogRecord{
						ctx:    ctx,
						target: tlh.target,
						record: record,
					},
				)
				return err
			}
		}
		return tlh.target.Handle(ctx, record)
	}
	tlh.base.backlog = append(
		tlh.base.backlog,
		backlogRecord{
			ctx:    ctx,
			target: tlh.target,
			record: record,
		},
	)
	return nil
}

// forwardBacklog sends the entire message backlog to the
// target log handler then clears the backlog.
// If an error is encountered when sending messages, unsent
// messages are preserved and an error is returned to indicate
// a retry on the next message submission
func (tlh *TriggeredLogHandler) forwardBacklog(ctx context.Context) error {
	for idx, record := range tlh.base.backlog {
		err := record.target.Handle(record.ctx, record.record)
		if err != nil {
			tlh.base.backlog = tlh.base.backlog[idx:]
			return err
		}
	}
	tlh.base.backlog = nil
	return nil
}

// WithAttrs returns a new TriggeredLogHandler with the provided
// attributes. The new handler shares the message backlog with
// it's parent, so triggering any handler in the tree will cause
// the entire backlog to be processed
func (tlh *TriggeredLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newTarget := tlh.target.WithAttrs(attrs)
	return &TriggeredLogHandler{
		target: newTarget,
		base:   tlh.base,
	}
}

// WithGroup returns a new TriggeredLogHandler with the provided
// group. The new handler shares the message backlog with
// it's parent, so triggering any handler in the tree will cause
// the entire backlog to be processed
func (tlh *TriggeredLogHandler) WithGroup(name string) slog.Handler {
	newTarget := tlh.target.WithGroup(name)
	return &TriggeredLogHandler{
		target: newTarget,
		base:   tlh.base,
	}
}
