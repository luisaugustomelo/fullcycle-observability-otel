package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type ZipCodeRequest struct {
	CEP string `json:"cep"`
}

var tracer = otel.Tracer("service-a")

func main() {
	initTracer()

	http.HandleFunc("/request-weather", requestWeatherHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Println("Service A running on port", port)
	http.ListenAndServe(":"+port, nil)
}

func requestWeatherHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "HandleRequestWeather")
	defer span.End()

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ZipCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidZip(req.CEP) {
		http.Error(w, `{"error":"invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	serviceBUrl := os.Getenv("SERVICE_B_URL")
	if serviceBUrl == "" {
		http.Error(w, "Service B URL not configured", http.StatusInternalServerError)
		return
	}

	req2, err := http.NewRequestWithContext(ctx, "GET", serviceBUrl+"/weather?cep="+req.CEP, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req2)
	if err != nil {
		http.Error(w, "Failed to contact Service B", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func isValidZip(zip string) bool {
	match, _ := regexp.MatchString(`^\d{8}$`, zip)
	return match
}

func initTracer() {
	endpoint := os.Getenv("ZIPKIN_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://zipkin:9411/api/v2/spans"
	}
	exporter, err := zipkin.New(endpoint)
	if err != nil {
		panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("service-a"),
		)),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
