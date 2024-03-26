# TriggeredLogHandler

I've made a short video introducing the library, if you prefer
that to reading: [https://youtu.be/GraNSbW4Dbk](https://youtu.be/GraNSbW4Dbk)

## The Problem

Traditional logging theory dictates that log messages should
specify a severity, and which messages are actually reported
should be adjusted based on a minimum severity.

The problem is that this model comes up short in the real
world. System failures are often preceeded by events that are
important to diagnosing the problem, and if those events are
reported at a lower severity, they won't be recorded, making
problem diagnosis more difficult.

The most common solution is to report all messages to a log
aggregation system, then use search tools to filter the messages
to find the critical information.

As systems scale, however, the infrastructure required to host a
sufficiently powerful log aggregation system becomes a problem.

## A Solution

TriggeredLogHandler is a go library designed to be used as an `slog.Handler`.
As a result, it integrates nicely with the logging you're already doing.

Log messages are orgnaized into "log streams", each with a unqiue ID to
allow them to be easily grouped and filtered.

Additionally, each log stream is configured with a trigger threshold. Log
messages are backlogged and only forwarded on if the log stream is triggered
by a message at or above the triggered severity threshold.

This allows the application to log as much as possible without overwhelming
log aggregation services. It also allows applications to be tested without
the need for any special logging setup.

## Example

```go
// This example shows the basics of using the logger in a
// web service
func Handle(w http.ResponseWriter, r *http.Request) {
    handler = triggeredloghandler.NewTriggeredLogHandler(
        slog.NewTextHandler(os.Stdout, nil),
        uuid.New().String(),
        slog.LevelError,
    )
    logger := slog.New(handler)
    // Now use the `logger` as any other logger. Messages will
    // only be printed to the console if one of the messages in
    // the log stream meets or exceeds the Error severity level.
}
```

See the examples directory for a more extensive example.

TriggeredLogHandler is safe for concurrent use.

## Shortcomings

In case it's not obvious, the library will increase memory usage
when an application does a lot of logging without triggering the
handler.

Experiments using the service in the examples directory show an
approximate 3x increase in garbage collector activity over a baseline
execution. However, this increase is on a service that allocates almost
no heap space on its own.

The performance impact is minimal enough to be difficult to measure reliably.

Using the service example, a
single thread was actually faster when using triggered logger than when
logging directly to /dev/null. I suspect this is due, in part, to the
fact that there are ample CPU cores to take care of garbage collection.

Submitting 16 simultaneous requests to the service example resulted in
a slowdown of 1 microsecond per request.

As a result, it's my belief that TriggeredLogHandler is appropriate for
production use. However, it's highly recommended that adopters test their
specific use cases to ensure performance is acceptable.

###### Install

```sh
go get github.com/williammoran/triggeredloghandler
```

###### 
