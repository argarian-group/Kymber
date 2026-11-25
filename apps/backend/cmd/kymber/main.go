package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const usage = `Usage:
  kymber login <panel-url> <api-token>   save the panel URL and token
  kymber apps                            list apps in every workspace
  kymber status <app-id>                 show an app's status
  kymber deploy [-version v] [-wait] <app-id>
                                         deploy an app; Git apps start a new build

KYMBER_URL and KYMBER_TOKEN override the saved login.
`

type config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type app struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Version     string `json:"image_version"`
	Status      string `json:"status"`
	LastError   string `json:"last_error"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	project     string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "kymber:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "login":
		if len(args) != 3 {
			return errors.New("usage: kymber login <panel-url> <api-token>")
		}
		return login(config{URL: strings.TrimRight(args[1], "/"), Token: args[2]})
	case "apps":
		return listApps()
	case "status":
		if len(args) != 2 {
			return errors.New("usage: kymber status <app-id>")
		}
		a, err := findApp(args[1])
		if err != nil {
			return err
		}
		printStatus(a)
		return nil
	case "deploy":
		return deploy(args[1:])
	default:
		fmt.Print(usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kymber", "config.json"), nil
}

func login(c config) error {
	var me struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := request(c, http.MethodGet, "/api/auth/me", nil, &me); err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, _ := json.Marshal(c)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	fmt.Printf("Signed in to %s as %s.\n", c.URL, me.User.Username)
	return nil
}

func loadConfig() (config, error) {
	var c config
	if path, err := configPath(); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			if err := json.Unmarshal(data, &c); err != nil {
				return c, fmt.Errorf("read %s: %w", path, err)
			}
		}
	}
	if url := os.Getenv("KYMBER_URL"); url != "" {
		c.URL = strings.TrimRight(url, "/")
	}
	if token := os.Getenv("KYMBER_TOKEN"); token != "" {
		c.Token = token
	}
	if c.URL == "" || c.Token == "" {
		return c, errors.New("not signed in; run kymber login <panel-url> <api-token>")
	}
	return c, nil
}

func request(c config, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.URL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &failure) == nil && failure.Error != "" {
			return errors.New(failure.Error)
		}
		return fmt.Errorf("%s %s: %s", method, path, res.Status)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func allApps(c config) ([]app, error) {
	var workspaces struct {
		Workspaces []struct {
			ID string `json:"id"`
		} `json:"workspaces"`
	}
	if err := request(c, http.MethodGet, "/api/workspaces", nil, &workspaces); err != nil {
		return nil, err
	}
	var apps []app
	for _, workspace := range workspaces.Workspaces {
		var projects struct {
			Projects []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"projects"`
		}
		if err := request(c, http.MethodGet, "/api/workspaces/"+workspace.ID+"/projects", nil, &projects); err != nil {
			return nil, err
		}
		for _, project := range projects.Projects {
			var services struct {
				Services []app `json:"services"`
			}
			if err := request(c, http.MethodGet, servicesPath(workspace.ID, project.ID), nil, &services); err != nil {
				return nil, err
			}
			for _, service := range services.Services {
				service.project = project.Name
				apps = append(apps, service)
			}
		}
	}
	return apps, nil
}

func servicesPath(workspaceID, projectID string) string {
	return "/api/workspaces/" + workspaceID + "/projects/" + projectID + "/services"
}

func listApps() error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	apps, err := allApps(c)
	if err != nil {
		return err
	}
	out := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(out, "ID\tNAME\tPROJECT\tSTATUS")
	for _, a := range apps {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.project, a.Status)
	}
	return out.Flush()
}

func findApp(id string) (app, error) {
	c, err := loadConfig()
	if err != nil {
		return app{}, err
	}
	apps, err := allApps(c)
	if err != nil {
		return app{}, err
	}
	for _, a := range apps {
		if a.ID == id || a.Name == id {
			return a, nil
		}
	}
	return app{}, fmt.Errorf("app %q not found", id)
}

func printStatus(a app) {
	fmt.Printf("%s (%s): %s\n", a.Name, a.ID, a.Status)
	if a.Status == "failed" && a.LastError != "" {
		fmt.Println(a.LastError)
	}
}

func deploy(args []string) error {
	flags := flag.NewFlagSet("deploy", flag.ContinueOnError)
	version := flags.String("version", "", "image version to deploy; defaults to the current one")
	wait := flags.Bool("wait", false, "wait until the deploy finishes")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: kymber deploy [-version v] [-wait] <app-id>")
	}
	c, err := loadConfig()
	if err != nil {
		return err
	}
	a, err := findApp(flags.Arg(0))
	if err != nil {
		return err
	}
	path := servicesPath(a.WorkspaceID, a.ProjectID) + "/" + a.ID
	var build struct {
		ID string `json:"id"`
	}
	if a.Image == "docker.io/kymber/build-artifacts" {
		if *version != "" {
			return errors.New("-version does not apply to Git apps; they deploy the build they start")
		}
		err = request(c, http.MethodPost, path+"/git-builds", nil, &build)
	} else {
		var body any
		if *version != "" {
			body = map[string]string{"image_version": *version}
		}
		err = request(c, http.MethodPost, path+"/deploy", body, nil)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Deploy of %s started.\n", a.Name)
	if !*wait {
		return nil
	}
	if build.ID != "" {
		if err := waitForBuild(c, path, build.ID); err != nil {
			return err
		}
	}
	for range time.Tick(3 * time.Second) {
		if err := request(c, http.MethodGet, path, nil, &a); err != nil {
			return err
		}
		if a.Status == "queued" || a.Status == "deploying" || build.ID != "" && a.Version != build.ID {
			continue
		}
		printStatus(a)
		if a.Status == "failed" {
			return errors.New("deploy failed")
		}
		return nil
	}
	return nil
}

func waitForBuild(c config, path, id string) error {
	fmt.Println("Building…")
	for range time.Tick(5 * time.Second) {
		var jobs []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Message string `json:"result_message"`
		}
		if err := request(c, http.MethodGet, path+"/git-builds", nil, &jobs); err != nil {
			return err
		}
		for _, job := range jobs {
			if job.ID != id {
				continue
			}
			switch job.Status {
			case "succeeded":
				return nil
			case "failed", "cancelled":
				return fmt.Errorf("build %s: %s", job.Status, job.Message)
			}
		}
	}
	return nil
}
