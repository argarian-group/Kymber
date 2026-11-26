package services

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const maxServiceDomains = 20

type serviceRoute struct {
	Host string
	Port int
}

// parseServiceRoutes reads the stored "host[:port],..." list; a host without a port uses appPort.
func parseServiceRoutes(value string, appPort int) []serviceRoute {
	var routes []serviceRoute
	for _, entry := range strings.Split(value, ",") {
		host, portText, _ := strings.Cut(strings.TrimSpace(entry), ":")
		if host == "" {
			continue
		}
		port := appPort
		if parsed, err := strconv.Atoi(portText); err == nil {
			port = parsed
		}
		routes = append(routes, serviceRoute{Host: host, Port: port})
	}
	return routes
}

func normalizeServiceDomains(value string, allowPorts bool) (string, error) {
	var entries []string
	seen := map[string]bool{}
	for _, entry := range strings.Split(strings.ToLower(value), ",") {
		host, portText, hasPort := strings.Cut(strings.TrimSpace(entry), ":")
		if host == "" && !hasPort {
			continue
		}
		if len(host) > 253 || !serviceDomainPattern.MatchString(host) || net.ParseIP(host) != nil {
			return "", fmt.Errorf("%q is not a valid public domain name", host)
		}
		if seen[host] {
			return "", fmt.Errorf("%s is listed twice", host)
		}
		seen[host] = true
		if hasPort {
			if !allowPorts {
				return "", errors.New("Compose stacks route every domain to their main service; remove the ports")
			}
			port, err := strconv.Atoi(portText)
			if err != nil || port < 1 || port > 65535 {
				return "", fmt.Errorf("the port for %s must be between 1 and 65535", host)
			}
			host += ":" + strconv.Itoa(port)
		}
		entries = append(entries, host)
	}
	if len(entries) > maxServiceDomains {
		return "", fmt.Errorf("an app can have at most %d domains", maxServiceDomains)
	}
	return strings.Join(entries, ","), nil
}

func traefikRouteLabels(serviceID string, routes []serviceRoute) []string {
	name := "kymber-" + serviceID
	labels := []string{
		"traefik.enable=true",
		"traefik.docker.network=kymber",
		"traefik.http.middlewares." + name + "-https.redirectscheme.scheme=https",
	}
	declared := map[int]bool{}
	for i, route := range routes {
		router := fmt.Sprintf("%s-%d", name, i)
		backend := fmt.Sprintf("%s-%d", name, route.Port)
		host := "Host(`" + route.Host + "`)"
		labels = append(labels,
			"traefik.http.routers."+router+"-http.rule="+host,
			"traefik.http.routers."+router+"-http.entrypoints=web",
			"traefik.http.routers."+router+"-http.middlewares="+name+"-https",
			"traefik.http.routers."+router+"-http.service="+backend,
			"traefik.http.routers."+router+".rule="+host,
			"traefik.http.routers."+router+".entrypoints=websecure",
			"traefik.http.routers."+router+".tls.certresolver=letsencrypt",
			"traefik.http.routers."+router+".service="+backend,
		)
		if !declared[route.Port] {
			declared[route.Port] = true
			labels = append(labels, fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", backend, route.Port))
		}
	}
	return labels
}

func domainHostRule(routes []serviceRoute) string {
	hosts := make([]string, len(routes))
	for i, route := range routes {
		hosts[i] = "Host(`" + route.Host + "`)"
	}
	return strings.Join(hosts, " || ")
}
