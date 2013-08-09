go-metrics
==========

This library provides the `metrics` package which can be used to instrument code and
expose application and runtime metrics in a flexible manner.

The `metrics` package exposes a few methods to emit stats which are delivered to a
configurable backends. Currently the following backends are supported:

* Statsite / Statsd

Examples
========

Here is an example of using the package:

    func SlowMethod() {
        // Profiling the runtime of a method
        start := time.Now()
        defer metrics.MeasureSince([]{"SlowMethod"}, start)
    }

    // Configure a statsite sink as the global metrics sink
    sink, _ := metrics.NewStatsiteSink("statsite:8125")
    metrics.NewGlobal(metrics.DefaultConfig("service-name"), sink)

    // Emit a Key/Value pair
    metrics.EmitKey([]string{"questions", "meaning of life"}, 42)

