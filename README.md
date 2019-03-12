go-metrics
==========

This library provides a `metrics` package which can be used to instrument code, expose application metrics, and profile runtime performance in a flexible manner.

[![CircleCI](https://circleci.com/gh/hugoluchessi/go-metrics.svg?style=svg)](https://circleci.com/gh/hugoluchessi/go-metrics) [![GoDoc]

Docs TDB

Sinks
-----

The `metrics` package makes use of a `MetricSink` interface to support delivery
to any type of backend. Currently the following sinks are provided:

* StatsdSink: Sinks to a [StatsD](https://github.com/etsy/statsd/) / statsite instance (UDP)
* PrometheusSink: Sinks to a [Prometheus](http://prometheus.io/) metrics endpoint (exposed via HTTP for scrapes)
* DatadogSink: Sinks to a [DataDog](https://www.datadoghq.com/) provider
* InmemSink: Provides in-memory aggregation, can be used to export stats
* FanoutSink: Sinks to multiple sinks. Enables writing to multiple statsite instances for example.
* BlackholeSink: Sinks to nowhere

In addition to the sinks, the `InmemSignal` can be used to catch a signal,
and dump a formatted output of recent metrics. For example, when a process gets
a SIGUSR1, it can dump to stderr recent performance metrics for debugging.

Labels
------

Most metrics do have an equivalent ending with `WithLabels`, such methods
allow to push metrics with labels and use some features of underlying Sinks
(ex: translated into Prometheus labels).

Examples
--------

Here is an example of using the package:

```go
func SlowMethod() {
    // Profiling the runtime of a method
    defer metrics.MeasureSince([]string{"SlowMethod"}, time.Now())
}

// Configure a statsite sink as the global metrics sink
sink, _ := metrics.NewStatsiteSink("statsite:8125")
metrics.NewGlobal(metrics.DefaultConfig("service-name"), sink)

// Emit a Key/Value pair
metrics.EmitKey([]string{"questions", "meaning of life"}, 42)
```

Here is an example of setting up a signal handler:

```go
// Setup the inmem sink and signal handler
inm := metrics.NewInmemSink(10*time.Second, time.Minute)
sig := metrics.DefaultInmemSignal(inm)
metrics.NewGlobal(metrics.DefaultConfig("service-name"), inm)

// Run some code
inm.SetGauge([]string{"foo"}, 42)
inm.EmitKey([]string{"bar"}, 30)

inm.IncrCounter([]string{"baz"}, 42)
inm.IncrCounter([]string{"baz"}, 1)
inm.IncrCounter([]string{"baz"}, 80)

inm.AddSample([]string{"method", "wow"}, 42)
inm.AddSample([]string{"method", "wow"}, 100)
inm.AddSample([]string{"method", "wow"}, 22)

....
```

When a signal comes in, output like the following will be dumped to stderr:

    [2014-01-28 14:57:33.04 -0800 PST][G] 'foo': 42.000
    [2014-01-28 14:57:33.04 -0800 PST][P] 'bar': 30.000
    [2014-01-28 14:57:33.04 -0800 PST][C] 'baz': Count: 3 Min: 1.000 Mean: 41.000 Max: 80.000 Stddev: 39.509
    [2014-01-28 14:57:33.04 -0800 PST][S] 'method.wow': Count: 3 Min: 22.000 Mean: 54.667 Max: 100.000 Stddev: 40.513