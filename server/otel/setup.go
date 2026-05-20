package otel

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	logglobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var shutdownFunc func(context.Context) error

func InitTelemetry(serviceName, version, commit, env string) func(context.Context) error {
	ctx := context.Background()

	hostname, _ := os.Hostname()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(env),
			semconv.HostName(hostname),
			semconv.ProcessPID(os.Getpid()),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithOSType(),
	)
	if err != nil {
		slog.Warn("Failed to create OTel resource", "error", err)
		res = resource.Default()
	}

	var tp *sdktrace.TracerProvider
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		slog.Error("Failed to create OTLP trace exporter", "error", err)
	} else {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
	}

	var mp *sdkmetric.MeterProvider
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		slog.Error("Failed to create OTLP metric exporter", "error", err)
	} else {
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(60*time.Second))),
		)
		otel.SetMeterProvider(mp)
	}

	var lp *sdklog.LoggerProvider
	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		slog.Error("Failed to create OTLP log exporter", "error", err)
	} else {
		lp = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		)
		logglobal.SetLoggerProvider(lp)
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))

	checkConnectivity(ctx)

	shutdownFunc = func(ctx context.Context) error {
		var errs []error
		if lp != nil {
			if err := lp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("log provider shutdown: %w", err))
			}
		}
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("metric provider shutdown: %w", err))
			}
		}
		if tp != nil {
			if err := tp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("trace provider shutdown: %w", err))
			}
		}
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}

	return shutdownFunc
}

func GetShutdownFunc() func(context.Context) error {
	return shutdownFunc
}

func checkConnectivity(ctx context.Context) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		slog.Warn("Invalid OTLP endpoint URL", "endpoint", endpoint, "error", err)
		return
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		slog.Warn("OTLP endpoint unreachable", "endpoint", endpoint, "error", err)
		return
	}
	conn.Close()
	slog.Info("OTLP endpoint reachable", "endpoint", endpoint)
}
