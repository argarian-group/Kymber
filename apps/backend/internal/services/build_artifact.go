package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kymber/backend/internal/artifacts"
)

func (h *Handler) importOCIImage(ctx context.Context, serverID, deploymentID string, source *os.File) (string, error) {
	if !entityIDPattern.MatchString(deploymentID) {
		return "", errors.New("invalid deployment id")
	}
	info, err := source.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > artifacts.MaxOCIArchiveSize {
		return "", errors.New("OCI archive exceeds 1 GiB")
	}
	tag := "docker.io/kymber/build-artifacts:" + deploymentID
	dockerArchive, expected, err := artifacts.ConvertOCIToDockerArchive(source, tag)
	if err != nil {
		return "", err
	}
	defer os.Remove(dockerArchive.Name())
	defer dockerArchive.Close()
	platformOutput, err := h.executeSSH(ctx, serverID, "docker info --format '{{.OSType}}/{{.Architecture}}'")
	if err != nil {
		return "", err
	}
	platformParts := strings.Split(strings.TrimSpace(string(platformOutput)), "/")
	if len(platformParts) != 2 || dockerPlatform(platformParts[0], platformParts[1]) != dockerPlatform(expected.OS, expected.Architecture) {
		return "", fmt.Errorf("build image platform %s/%s does not match server platform %s", expected.OS, expected.Architecture, strings.TrimSpace(string(platformOutput)))
	}
	command := "set -eu; docker image load --quiet >/dev/null; docker image inspect --format '{{json .}}' " + shellQuote(tag)
	output, err := h.executeSSHStream(ctx, serverID, command, dockerArchive)
	if err != nil {
		return "", err
	}
	var imported struct {
		ID           string          `json:"Id"`
		OS           string          `json:"Os"`
		Architecture string          `json:"Architecture"`
		Config       json.RawMessage `json:"Config"`
		RootFS       struct {
			Layers []string `json:"Layers"`
		} `json:"RootFS"`
	}
	if err := json.Unmarshal(output, &imported); err != nil {
		return "", fmt.Errorf("could not verify imported image: %w", err)
	}
	if imported.ID == "" || imported.OS != expected.OS || imported.Architecture != expected.Architecture || !sameJSON(imported.Config, expected.Config) || !sameStrings(imported.RootFS.Layers, expected.DiffIDs) {
		return "", errors.New("imported image does not match the validated OCI artifact")
	}
	return imported.ID, nil
}

func (h *Handler) importGitBuildArtifact(ctx context.Context, serverID, serviceID, jobID string) error {
	var artifactPath string
	if err := h.db.QueryRowContext(ctx, `SELECT artifact_path FROM git_build_jobs WHERE id = ? AND service_id = ? AND status = 'succeeded'
		AND artifact_expires_at > UTC_TIMESTAMP(6)`, jobID, serviceID).Scan(&artifactPath); err != nil {
		return errors.New("no unexpired successful Git build artifact is available")
	}
	if filepath.Base(artifactPath) != artifactPath || artifactPath == "." {
		return errors.New("invalid stored Git build artifact path")
	}
	directory := os.Getenv("KYMBER_BUILD_ARTIFACT_DIR")
	if directory == "" {
		return errors.New("build artifact storage is not configured")
	}
	archive, err := os.Open(filepath.Join(directory, artifactPath))
	if err != nil {
		return err
	}
	defer archive.Close()
	_, err = h.importOCIImage(ctx, serverID, jobID, archive)
	return err
}

func dockerPlatform(osName, architecture string) string {
	switch architecture {
	case "x86_64":
		architecture = "amd64"
	case "aarch64":
		architecture = "arm64"
	}
	return osName + "/" + architecture
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
