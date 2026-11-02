package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Point struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

var ranges = map[string]time.Duration{"1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour}

var ErrUnavailable = errors.New("metrics are unavailable")

var client = &http.Client{Timeout: 15 * time.Second}

func ParseRange(value string) (time.Duration, bool) {
	if value == "" {
		return time.Hour, true
	}
	span, ok := ranges[value]
	return span, ok
}

func prometheusURL() string {
	if value := os.Getenv("KYMBER_PROMETHEUS_URL"); value != "" {
		return value
	}
	return "http://prometheus:9090"
}

func get(ctx context.Context, path string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, prometheusURL()+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: prometheus returned %d", ErrUnavailable, response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func queryRange(ctx context.Context, query string, span time.Duration) ([]Point, error) {
	end := time.Now()
	step := max(span/120, 15*time.Second)
	var body struct {
		Data struct {
			Result []struct {
				Values [][2]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	values := url.Values{"query": {query}, "start": {strconv.FormatInt(end.Add(-span).Unix(), 10)}, "end": {strconv.FormatInt(end.Unix(), 10)}, "step": {strconv.Itoa(int(step.Seconds()))}}
	if err := get(ctx, "/api/v1/query_range", values, &body); err != nil {
		return nil, err
	}
	points := make([]Point, 0)
	if len(body.Data.Result) == 0 {
		return points, nil
	}
	for _, pair := range body.Data.Result[0].Values {
		timestamp, _ := pair[0].(float64)
		text, _ := pair[1].(string)
		value, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		points = append(points, Point{T: int64(timestamp), V: value})
	}
	return points, nil
}

func QueryAll(ctx context.Context, queries map[string]string, span time.Duration) (map[string][]Point, error) {
	series := make(map[string][]Point, len(queries))
	for name, query := range queries {
		points, err := queryRange(ctx, query, span)
		if err != nil {
			return nil, err
		}
		series[name] = points
	}
	return series, nil
}

type firingAlert struct {
	Name      string
	ServerID  string
	ServiceID string
	Since     time.Time
}

func firingAlerts(ctx context.Context) ([]firingAlert, error) {
	var body struct {
		Data struct {
			Alerts []struct {
				Labels   map[string]string `json:"labels"`
				State    string            `json:"state"`
				ActiveAt time.Time         `json:"activeAt"`
			} `json:"alerts"`
		} `json:"data"`
	}
	if err := get(ctx, "/api/v1/alerts", url.Values{}, &body); err != nil {
		return nil, err
	}
	alerts := make([]firingAlert, 0)
	for _, alert := range body.Data.Alerts {
		if alert.State == "firing" {
			alerts = append(alerts, firingAlert{Name: alert.Labels["alertname"], ServerID: alert.Labels["server_id"], ServiceID: alert.Labels["service_id"], Since: alert.ActiveAt})
		}
	}
	return alerts, nil
}
