package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

var tracer = otel.Tracer("service-b")

func main() {
	initTracer()

	http.HandleFunc("/weather", weatherHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Service B running on port", port)
	http.ListenAndServe(":"+port, nil)
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "HandleWeather")
	defer span.End()

	zip := r.URL.Query().Get("cep")
	if !isValidZip(zip) {
		http.Error(w, `{"error":"invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	city, err := getCityByZip(ctx, zip)
	if err != nil {
		http.Error(w, `{"error":"can not find zipcode"}`, http.StatusNotFound)
		return
	}

	tempC, err := getTemperature(ctx, city)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch temperature"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"city":   city,
		"temp_C": tempC,
		"temp_F": tempC*1.8 + 32,
		"temp_K": tempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func isValidZip(zip string) bool {
	match, _ := regexp.MatchString(`^\d{8}$`, zip)
	return match
}

func getCityByZip(ctx context.Context, zip string) (string, error) {
	ctx, span := tracer.Start(ctx, "getCityByZip")
	defer span.End()

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://viacep.com.br/ws/"+zip+"/json/", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("invalid zip")
	}

	var data ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Localidade == "" {
		return "", errors.New("city not found")
	}

	return data.Localidade, nil
}

func getTemperature(ctx context.Context, city string) (float64, error) {
	ctx, span := tracer.Start(ctx, "getTemperature")
	defer span.End()

	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return 0, errors.New("weather api key not configured")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", apiKey, city)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, errors.New("failed to fetch weather")
	}

	var data WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return data.Current.TempC, nil
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
			semconv.ServiceName("service-b"),
		)),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
