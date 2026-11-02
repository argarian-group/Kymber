package monitoring

import (
	"regexp"
	"strings"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func fill(template, id string) string {
	return strings.ReplaceAll(template, "ID", id)
}

func ServerQueries(serverID string) (map[string]string, bool) {
	if !idPattern.MatchString(serverID) {
		return nil, false
	}
	return map[string]string{
		"cpu":         fill(`100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle",server_id="ID"}[5m])))`, serverID),
		"memory":      fill(`100 * (1 - sum(node_memory_MemAvailable_bytes{server_id="ID"}) / sum(node_memory_MemTotal_bytes{server_id="ID"}))`, serverID),
		"disk":        fill(`100 * (1 - sum(node_filesystem_avail_bytes{server_id="ID",mountpoint="/"}) / sum(node_filesystem_size_bytes{server_id="ID",mountpoint="/"}))`, serverID),
		"network_in":  fill(`sum(rate(node_network_receive_bytes_total{server_id="ID",device!~"lo|veth.*|docker.*|br-.*"}[5m]))`, serverID),
		"network_out": fill(`sum(rate(node_network_transmit_bytes_total{server_id="ID",device!~"lo|veth.*|docker.*|br-.*"}[5m]))`, serverID),
	}, true
}

func AppQueries(serviceID string, withHTTP bool) (map[string]string, bool) {
	if !idPattern.MatchString(serviceID) {
		return nil, false
	}
	queries := map[string]string{
		"cpu":         fill(`100 * sum(rate(container_cpu_usage_seconds_total{service_id="ID"}[5m]))`, serviceID),
		"memory":      fill(`sum(container_memory_working_set_bytes{service_id="ID"})`, serviceID),
		"network_in":  fill(`sum(rate(container_network_receive_bytes_total{service_id="ID"}[5m]))`, serviceID),
		"network_out": fill(`sum(rate(container_network_transmit_bytes_total{service_id="ID"}[5m]))`, serviceID),
	}
	if withHTTP {
		queries["requests"] = fill(`sum(rate(traefik_router_requests_total{service_id="ID"}[5m]))`, serviceID)
		queries["errors"] = fill(`100 * (sum(rate(traefik_router_requests_total{service_id="ID",code=~"5.."}[5m])) or vector(0)) / sum(rate(traefik_router_requests_total{service_id="ID"}[5m]))`, serviceID)
		queries["latency_p95"] = fill(`histogram_quantile(0.95, sum by (le) (rate(traefik_router_request_duration_seconds_bucket{service_id="ID"}[5m])))`, serviceID)
	}
	return queries, true
}
