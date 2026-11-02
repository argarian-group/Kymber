package services

import (
	"strings"
	"testing"
)

func TestValidatePort(t *testing.T) {
	cases := []struct {
		host, container int
		protocol, bind  string
		ok              bool
	}{
		{8080, 80, "tcp", "0.0.0.0", true},
		{53, 53, "udp", "127.0.0.1", true},
		{0, 80, "tcp", "0.0.0.0", false},
		{70000, 80, "tcp", "0.0.0.0", false},
		{8080, 0, "tcp", "0.0.0.0", false},
		{8080, 70000, "tcp", "0.0.0.0", false},
		{8080, 80, "sctp", "0.0.0.0", false},
		{8080, 80, "tcp", "10.0.0.1", false},
		{8080, 80, "tcp", "", false},
	}
	for _, c := range cases {
		if err := validatePort(c.host, c.container, c.protocol, c.bind); (err == nil) != c.ok {
			t.Errorf("%+v: err = %v", c, err)
		}
	}
}

func TestPortDeployCommand(t *testing.T) {
	check, args := portDeployCommand([]servicePort{{HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Bind: "0.0.0.0"}, {HostPort: 53, ContainerPort: 5353, Protocol: "udp", Bind: "127.0.0.1"}})
	if args != " --publish 0.0.0.0:8080:80/tcp --publish 127.0.0.1:53:5353/udp" {
		t.Errorf("args = %q", args)
	}
	for _, want := range []string{"ss -Hltn 'sport = :8080'", "ss -Hlun 'sport = :53'", `docker port "$n"`, "exit 1"} {
		if !strings.Contains(check, want) {
			t.Errorf("check lacks %q: %s", want, check)
		}
	}
	if check, args := portDeployCommand(nil); check != "" || args != "" {
		t.Errorf("no ports: %q %q", check, args)
	}
}
