package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests",
		},
		[]string{"path", "method"},
	)
)

type WeatherResponse struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
}

func getTemperatureFromAPI() (float64, error) {
	// 🔁 Для демо используем случайную температуру (чтобы не зависеть от API-ключа)
	// Если хочешь реальный API — раскомментируй ниже и вставь ключ
	// return realAPICall()

	rand.Seed(time.Now().UnixNano())
	return 15.0 + rand.Float64()*20.0, nil // 15–35°C
}

// func realAPICall() (float64, error) {
// 	apiKey := "YOUR_API_KEY_HERE"
// 	city := "London"
// 	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, apiKey)
//
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer resp.Body.Close()
//
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return 0, err
// 	}
//
// 	var weather WeatherResponse
// 	if err := json.Unmarshal(body, &weather); err != nil {
// 		return 0, err
// 	}
// 	return weather.Main.Temp, nil
// }

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(requestDuration.WithLabelValues(r.URL.Path, r.Method))
	defer timer.ObserveDuration()

	temp, err := getTemperatureFromAPI()
	if err != nil {
		log.Printf("Error fetching temperature: %v", err)
		temp = 0
	}

	// Определяем статус ДО отправки тела
	status := http.StatusOK
	if rand.Float64() < 0.05 { // 5% ошибок
		status = http.StatusInternalServerError
	}

	// Подготавливаем ответ
	response := map[string]interface{}{
		"temperature": temp,
		"unit":        "°C",
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	if status != http.StatusOK {
		response["error"] = "internal server error"
	}

	// Устанавливаем заголовки и статус
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Отправляем тело
	json.NewEncoder(w).Encode(response)

	// Инкрементируем метрику с правильным статусом
	requestsTotal.WithLabelValues(r.URL.Path, r.Method, fmt.Sprintf("%d", status)).Inc()
}

func main() {
	// Регистрируем стандартные метрики (чтобы /metrics не был пустым при старте)
	prometheus.MustRegister(prometheus.NewGoCollector())
	prometheus.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	http.HandleFunc("/weather", weatherHandler)
	http.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	fmt.Printf("🚀 Server starting on %s\n", addr)
	fmt.Printf("✅ Test endpoints:\n")
	fmt.Printf("   - http://localhost:8080/weather\n")
	fmt.Printf("   - http://localhost:8080/metrics\n")
	log.Fatal(http.ListenAndServe(addr, nil))
}
