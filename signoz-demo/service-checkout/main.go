package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// client membungkus transport OTel agar trace context otomatis diteruskan
// ke payment-service lewat HTTP header — inilah yang membuat tracing "distributed".
var client = &http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
}

func main() {
	ctx := context.Background()

	shutdown, err := setupOTel(ctx, "checkout-service")
	if err != nil {
		log.Fatalf("otel init: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	mux := http.NewServeMux()
	// otelhttp.NewHandler otomatis membuat span untuk setiap request masuk
	mux.Handle("/checkout", otelhttp.NewHandler(http.HandlerFunc(handleCheckout), "GET /checkout"))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := getenv("PORT", "8081")
	server := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("checkout-service listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	paymentURL := getenv("PAYMENT_SERVICE_URL", "http://localhost:8082") + "/process"

	// NewRequestWithContext meneruskan span context ke HTTP client
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, paymentURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// client.Do menyisipkan W3C Trace Context header secara otomatis
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// setupOTel menginisialisasi OpenTelemetry tracer dengan OTLP HTTP exporter ke SigNoz.
func setupOTel(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	traceURL := strings.TrimRight(endpoint, "/") + "/v1/traces"

	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(traceURL))
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(getenv("OTEL_DEPLOYMENT_ENVIRONMENT", "demo")),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	// TraceContext propagator agar span context bisa dibaca oleh service lain via HTTP header
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Printf("OTel → SigNoz: %s  service=%s", traceURL, serviceName)
	return tp.Shutdown, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
