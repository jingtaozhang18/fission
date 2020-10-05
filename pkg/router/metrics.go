package router

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var globalFunctionCallCount uint64

type (
	// functionLabels is the set of metrics labels that relate to
	// functions.
	//
	// cached indicates whether or not the function call hit the
	// cache in this service.
	//
	// namespace and name are the metadata of the function.
	functionLabels struct {
		cached    bool
		namespace string
		name      string
	}

	// httpLabels is the set of metrics labels that relate to HTTP
	// requests.
	//
	// host is the host that the HTTP request was made to
	// path is the relative URL of the request
	// method is the HTTP method ("GET", "POST", ...)
	// code is the HTTP status code
	// source the source of request
	// stype the type of source
	httpLabels struct {
		host   string
		path   string
		method string
		code   int
		source string
		stype  string
	}
)

var (
	metricAddr = ":8080"

	// function + http labels as strings
	labelsStrings = []string{"cached", "namespace", "name", "host", "path", "method", "code"}

	// Function http calls count
	// cached: true | false, is this function service address cached locally
	// namespace: function namespace
	// name: function name
	// code: http status code
	// path: the client call the function on which http path
	// method: the function's http method

	// fission flow recorder labels
	flowLabelStrings = []string{"source", "destination", "stype", "dtype", "method", "code"}
	// source: "func.{namespace}.{name}" or "{kafka|nats|azurequeue|...}.{topic}"
	// destination: "func.{namespace}.{name}" or "{kafka|nats|azurequeue|...}.{topic}"
	// stype: the type of source
	// dtype: the type of destination
	// method: the function's http method
	// code: http status code
	functionCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_function_calls_total",
			Help: "Count of Fission function calls",
		},
		labelsStrings,
	)
	functionCallErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_function_errors_total",
			Help: "Count of Fission function errors",
		},
		labelsStrings,
	)
	functionCallDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_function_duration_seconds",
			Help:       "Runtime duration of the Fission function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)
	functionCallOverhead = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_function_overhead_seconds",
			Help:       "The function call delay caused by fission.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)
	functionCallResponseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_function_response_size_bytes",
			Help:       "The response size of the http call to target function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)
	fissionFlowRecorder = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_flow_recorder_by_router",
			Help: "recorder of network flow",
		},
		flowLabelStrings,
	)
)

func init() {
	prometheus.MustRegister(functionCalls)
	prometheus.MustRegister(functionCallErrors)
	prometheus.MustRegister(functionCallDuration)
	prometheus.MustRegister(functionCallOverhead)
	prometheus.MustRegister(functionCallResponseSize)
	prometheus.MustRegister(fissionFlowRecorder)
}

func labelsToStrings(f *functionLabels, h *httpLabels) []string {
	var cached string
	if f.cached {
		cached = "true"
	} else {
		cached = "false"
	}
	return []string{
		cached,
		f.namespace,
		f.name,
		h.host,
		h.path,
		h.method,
		fmt.Sprint(h.code),
	}
}

func flowLabelsToStrings(f *functionLabels, h *httpLabels) []string {
	// "source", "destination", "stype", "dtype", "method", "code"
	return []string{
		h.source,
		fmt.Sprintf("func.%s.%s", f.namespace, f.name),
		h.stype,
		"func",
		h.method,
		fmt.Sprint(h.code),
	}
}

func functionCallCompleted(f *functionLabels, h *httpLabels, overhead, duration time.Duration, respSize int64) {
	atomic.AddUint64(&globalFunctionCallCount, 1)

	l := labelsToStrings(f, h)
	fl := flowLabelsToStrings(f, h)

	// recorder flow
	fissionFlowRecorder.WithLabelValues(fl...).Inc()

	// overhead: time from request ingress into router upto proxing into function pod
	functionCallOverhead.WithLabelValues(l...).Observe(float64(overhead.Nanoseconds()) / 1e9)

	// total function call counter
	functionCalls.WithLabelValues(l...).Inc()

	// error counter
	if h.code >= 400 {
		functionCallErrors.WithLabelValues(l...).Inc()
	}

	// duration summary
	functionCallDuration.WithLabelValues(l...).Observe(float64(duration.Nanoseconds()) / 1e9)

	// Response size.  -1 means the size unknown, in which case we don't report it.
	if respSize != -1 {
		functionCallResponseSize.WithLabelValues(l...).Observe(float64(respSize))
	}
}
