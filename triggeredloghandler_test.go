package triggeredloghandler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	target := &mockHandler{}
	const streamID = "STREAM_ID"
	const triggerLevel = slog.LevelDebug
	tlh := NewTriggeredLogHandler(target, streamID, triggerLevel)
	if tlh.base.triggerLevel != triggerLevel {
		t.Errorf(
			"Expect trigger level %s but %s",
			triggerLevel.String(),
			tlh.base.triggerLevel.String(),
		)
	}
	if len(target.attrs) != 1 {
		t.Errorf("Expect 1 attribute but %+v", target.attrs)
	} else {
		if target.attrs[0].Key != TriggeredLogStreamIDKey {
			t.Errorf(
				"Expect attribute key %s but %s",
				TriggeredLogStreamIDKey,
				target.attrs[0].Key,
			)
		}
		if target.attrs[0].Value.String() != streamID {
			t.Errorf(
				"Expect attribute value %s but %s",
				streamID,
				target.attrs[0].Value.String(),
			)
		}
	}
}

func TestWithAttrs(t *testing.T) {
	target := &mockHandler{}
	const streamID = "STREAM_ID"
	const triggerLevel = slog.LevelDebug
	const attrKey = "TEST_ATTR_KEY"
	const attrValue = "TEST_ATTR_VALUE"
	tlh := NewTriggeredLogHandler(target, streamID, triggerLevel)
	tlh2 := tlh.WithAttrs(
		[]slog.Attr{{Key: attrKey, Value: slog.StringValue(attrValue)}},
	).(*TriggeredLogHandler)
	if tlh.base != tlh2.base {
		t.Error("Base data was altered by WithAttrs()")
	}
	if len(target.attrs) != 1 {
		t.Errorf("Expect 1 attribute but %+v", target.attrs)
	} else {
		if target.attrs[0].Key != attrKey {
			t.Errorf("Expect key %s but got %s", attrKey, target.attrs[0].Key)
		}
		if target.attrs[0].Value.String() != attrValue {
			t.Errorf(
				"Expect value %s but got %s",
				attrValue,
				target.attrs[0].Value.String(),
			)
		}
	}
}

func TestWithGroup(t *testing.T) {
	target := &mockHandler{}
	const streamID = "STREAM_ID"
	const triggerLevel = slog.LevelDebug
	const group = "GROUP_NAME"
	tlh := NewTriggeredLogHandler(target, streamID, triggerLevel)
	tlh2 := tlh.WithGroup(group).(*TriggeredLogHandler)
	if tlh.base != tlh2.base {
		t.Error("Base data was altered by WithGroup()")
	}
	if target.group != group {
		t.Errorf("Expected group to be %s but %s", group, target.group)
	}
}

func TestHandle(t *testing.T) {
	target := &mockHandler{}
	const streamID = "STREAM_ID"
	const triggerLevel = slog.LevelWarn
	tlh := NewTriggeredLogHandler(target, streamID, triggerLevel)
	type contextType string
	const contextKey = contextType("CONTEXT_KEY")
	const contextValue1 = "CONTEXT_VALUE_1"
	ctx := context.WithValue(context.Background(), contextKey, contextValue1)
	const message1 = "MESSAGE 1"
	record := slog.NewRecord(time.Now(), slog.LevelDebug, message1, 0)
	err := tlh.Handle(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.records) != 0 {
		t.Fatalf("No records should be logged but %+v", target.records)
	}
	if len(tlh.base.backlog) != 1 {
		t.Fatalf("Expect 1 record in backlog but %+v", tlh.base.backlog)
	}
	if tlh.base.backlog[0].record.Message != message1 {
		t.Fatalf(
			"Expect message %s but %s",
			message1,
			tlh.base.backlog[0].record.Message,
		)
	}
	const contextValue2 = "CONTEXT_VALUE_2"
	ctx = context.WithValue(context.Background(), contextKey, contextValue2)
	const message2 = "MESSAGE 2"
	record = slog.NewRecord(time.Now(), slog.LevelError, message2, 0)
	err = tlh.Handle(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if len(tlh.base.backlog) != 0 {
		t.Fatalf("Expect empty backlog but %+v", tlh.base.backlog)
	}
	if len(target.records) != 2 {
		t.Fatalf("Should have logged 2 records but %+v", target.records)
	}
	if target.records[0].record.Message != message1 {
		t.Errorf(
			"Record 0 should be %s but %s",
			message1,
			target.records[0].record.Message,
		)
	}
	if target.records[0].ctx.Value(contextKey) != contextValue1 {
		t.Errorf(
			"Record 0 should have context value %s but %s",
			contextValue1,
			target.records[0].ctx.Value(contextKey),
		)
	}
	if target.records[1].record.Message != message2 {
		t.Errorf(
			"Record 1 should be %s but %s",
			message2,
			target.records[1].record.Message,
		)
	}
	if target.records[1].ctx.Value(contextKey) != contextValue2 {
		t.Errorf(
			"Record 1 should have context value %s but %s",
			contextValue2,
			target.records[1].ctx.Value(contextKey),
		)
	}
}

func TestFailurePreservesMessages(t *testing.T) {
	target := &mockHandler{}
	const streamID = "STREAM_ID"
	const triggerLevel = slog.LevelWarn
	tlh := NewTriggeredLogHandler(target, streamID, triggerLevel)
	const message1 = "MESSAGE 1"
	record := slog.NewRecord(time.Now(), slog.LevelDebug, message1, 0)
	err := tlh.Handle(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if len(tlh.base.backlog) != 1 {
		t.Fatalf("Should have 1 record in backlog, but %+v", tlh.base.backlog)
	}
	target.handleError = true
	const message2 = "MESSAGE 2"
	record = slog.NewRecord(time.Now(), slog.LevelError, message2, 0)
	err = tlh.Handle(context.Background(), record)
	if err == nil {
		t.Fatal("Error was not bubbled up")
	}
	if len(tlh.base.backlog) != 2 {
		t.Fatalf("Should have 2 records in backlog, but %+v", tlh.base.backlog)
	}
	target.handleError = false
	const message3 = "MESSAGE 3"
	record = slog.NewRecord(time.Now(), slog.LevelError, message3, 0)
	err = tlh.Handle(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if len(tlh.base.backlog) != 0 {
		t.Fatalf("backlog should be empty, but %+v", tlh.base.backlog)
	}
	if len(target.records) != 3 {
		t.Fatalf("Should be 3 logged records, but %+v", target.records)
	}
}

type mockHandler struct {
	handleError bool
	attrs       []slog.Attr
	group       string
	records     []logRecord
}

type logRecord struct {
	ctx    context.Context
	record slog.Record
}

func (mh *mockHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (mh *mockHandler) Handle(ctx context.Context, record slog.Record) error {
	if mh.handleError {
		return errors.New("mock handler error")
	}
	mh.records = append(mh.records, logRecord{ctx, record})
	return nil
}

func (mh *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	mh.attrs = attrs
	return mh
}

func (mh *mockHandler) WithGroup(name string) slog.Handler {
	mh.group = name
	return mh
}
