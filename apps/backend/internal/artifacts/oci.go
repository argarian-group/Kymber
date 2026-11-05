package artifacts

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const MaxOCIArchiveSize int64 = 1 << 30
const maxOCIMetadataSize int64 = 4 << 20

var sha256Digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var dockerTag = regexp.MustCompile(`^[a-z0-9]+(?:[._-]+[a-z0-9]+)*(?:/[a-z0-9]+(?:[._-]+[a-z0-9]+)*)+:[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

type ociLayout struct {
	Version string `json:"imageLayoutVersion"`
}

type ociIndex struct {
	SchemaVersion int             `json:"schemaVersion"`
	Manifests     []ociDescriptor `json:"manifests"`
}

type ociManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	Config        ociDescriptor   `json:"config"`
	Layers        []ociDescriptor `json:"layers"`
}

type ociDescriptor struct {
	Digest string `json:"digest"`
}

type dockerManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

type OCIImage struct {
	OS           string
	Architecture string
	Config       json.RawMessage
	DiffIDs      []string
}

type ociImageConfig struct {
	OS           string          `json:"os"`
	Architecture string          `json:"architecture"`
	Config       json.RawMessage `json:"config"`
	RootFS       struct {
		Type    string   `json:"type"`
		DiffIDs []string `json:"diff_ids"`
	} `json:"rootfs"`
}

func SpoolOCI(r io.Reader) (*os.File, string, error) {
	file, err := os.CreateTemp("", "kymber-oci-*")
	if err != nil {
		return nil, "", err
	}
	remove := func(err error) (*os.File, string, error) {
		file.Close()
		os.Remove(file.Name())
		return nil, "", err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(r, MaxOCIArchiveSize+1))
	if err != nil {
		return remove(err)
	}
	if size > MaxOCIArchiveSize {
		return remove(errors.New("OCI archive exceeds 1 GiB"))
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return remove(err)
	}
	if err := validateOCI(file, size); err != nil {
		return remove(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return remove(err)
	}
	return file, hex.EncodeToString(hash.Sum(nil)), nil
}

func ConvertOCIToDockerArchive(oci *os.File, tag string) (*os.File, OCIImage, error) {
	if !dockerTag.MatchString(tag) {
		return nil, OCIImage{}, errors.New("invalid image tag")
	}
	info, err := oci.Stat()
	if err != nil {
		return nil, OCIImage{}, err
	}
	if info.Size() > MaxOCIArchiveSize {
		return nil, OCIImage{}, errors.New("OCI archive exceeds 1 GiB")
	}
	if err := validateOCI(oci, info.Size()); err != nil {
		return nil, OCIImage{}, err
	}
	indexData, err := readOCIEntry(oci, info.Size(), "index.json", maxOCIMetadataSize)
	if err != nil {
		return nil, OCIImage{}, err
	}
	var index ociIndex
	if err := json.Unmarshal(indexData, &index); err != nil || len(index.Manifests) != 1 {
		return nil, OCIImage{}, errors.New("OCI archive must contain exactly one image manifest")
	}
	manifestDigest := strings.TrimPrefix(index.Manifests[0].Digest, "sha256:")
	manifestData, err := readOCIEntry(oci, info.Size(), "blobs/sha256/"+manifestDigest, maxOCIMetadataSize)
	if err != nil {
		return nil, OCIImage{}, err
	}
	var image ociManifest
	if err := json.Unmarshal(manifestData, &image); err != nil {
		return nil, OCIImage{}, errors.New("invalid OCI image manifest")
	}
	configDigest := strings.TrimPrefix(image.Config.Digest, "sha256:")
	configName := configDigest + ".json"
	configData, err := readOCIEntry(oci, info.Size(), "blobs/sha256/"+configDigest, maxOCIMetadataSize)
	if err != nil {
		return nil, OCIImage{}, err
	}
	var config ociImageConfig
	if err := json.Unmarshal(configData, &config); err != nil || config.OS == "" || config.Architecture == "" || config.RootFS.Type != "layers" || len(config.RootFS.DiffIDs) != len(image.Layers) {
		return nil, OCIImage{}, errors.New("invalid OCI image configuration")
	}
	for _, diffID := range config.RootFS.DiffIDs {
		if !validDigest(diffID) {
			return nil, OCIImage{}, errors.New("invalid OCI layer diff ID")
		}
	}
	layers := make([]string, len(image.Layers))
	for i := range layers {
		layers[i] = "layer-" + strconv.Itoa(i) + "/layer.tar"
	}
	manifestJSON, err := json.Marshal([]dockerManifest{{Config: configName, RepoTags: []string{tag}, Layers: layers}})
	if err != nil {
		return nil, OCIImage{}, err
	}
	output, err := os.CreateTemp("", "kymber-docker-image-*")
	if err != nil {
		return nil, OCIImage{}, err
	}
	remove := func(err error) (*os.File, OCIImage, error) {
		output.Close()
		os.Remove(output.Name())
		return nil, OCIImage{}, err
	}
	limited := &limitedWriter{writer: output, limit: MaxOCIArchiveSize}
	tarWriter := tar.NewWriter(limited)
	writeEntry := func(name string, data []byte) error {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err := tarWriter.Write(data)
		return err
	}
	if err := writeEntry("manifest.json", manifestJSON); err != nil {
		return remove(err)
	}
	if err := writeEntry(configName, configData); err != nil {
		return remove(err)
	}
	for i, layer := range image.Layers {
		blobName := "blobs/sha256/" + strings.TrimPrefix(layer.Digest, "sha256:")
		if err := writeOCIEntry(oci, info.Size(), blobName, "layer-"+strconv.Itoa(i)+"/layer.tar", tarWriter); err != nil {
			return remove(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		return remove(err)
	}
	if _, err := output.Seek(0, io.SeekStart); err != nil {
		return remove(err)
	}
	return output, OCIImage{OS: config.OS, Architecture: config.Architecture, Config: config.Config, DiffIDs: config.RootFS.DiffIDs}, nil
}

type limitedWriter struct {
	writer  io.Writer
	limit   int64
	written int64
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.limit-w.written {
		return 0, errors.New("Docker archive exceeds 1 GiB")
	}
	n, err := w.writer.Write(data)
	w.written += int64(n)
	return n, err
}

func readOCIEntry(file *os.File, archiveSize int64, name string, limit int64) ([]byte, error) {
	var data []byte
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	reader := tar.NewReader(io.LimitReader(file, archiveSize))
	for {
		header, err := reader.Next()
		if err != nil {
			return nil, errors.New("OCI archive entry not found")
		}
		if header.Name != name {
			continue
		}
		if header.Size > limit {
			return nil, errors.New("OCI entry exceeds size limit")
		}
		data, err = io.ReadAll(io.LimitReader(reader, limit+1))
		if err != nil || int64(len(data)) != header.Size {
			return nil, errors.New("OCI entry is truncated")
		}
		return data, nil
	}
}

func writeOCIEntry(file *os.File, archiveSize int64, sourceName, targetName string, writer *tar.Writer) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	reader := tar.NewReader(io.LimitReader(file, archiveSize))
	for {
		header, err := reader.Next()
		if err != nil {
			return errors.New("OCI layer not found")
		}
		if header.Name != sourceName {
			continue
		}
		if err := writer.WriteHeader(&tar.Header{Name: targetName, Mode: 0600, Size: header.Size, Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		written, err := io.CopyN(writer, reader, header.Size)
		if err != nil || written != header.Size {
			return errors.New("OCI layer is truncated")
		}
		return nil
	}
}

func validateOCI(file *os.File, archiveSize int64) error {
	blobs := make(map[string]struct{})
	var layoutData, indexData []byte
	if err := scanOCI(file, archiveSize, func(name string, data []byte) error {
		switch name {
		case "oci-layout":
			layoutData = data
		case "index.json":
			indexData = data
		default:
			blobs[strings.TrimPrefix(name, "blobs/sha256/")] = struct{}{}
		}
		return nil
	}); err != nil {
		return err
	}
	var layout ociLayout
	if json.Unmarshal(layoutData, &layout) != nil || layout.Version != "1.0.0" {
		return errors.New("invalid OCI layout metadata")
	}
	var index ociIndex
	if json.Unmarshal(indexData, &index) != nil || index.SchemaVersion != 2 || len(index.Manifests) == 0 {
		return errors.New("invalid OCI image index")
	}
	for _, manifest := range index.Manifests {
		if !validDigest(manifest.Digest) {
			return errors.New("invalid OCI manifest digest")
		}
		manifestDigest := strings.TrimPrefix(manifest.Digest, "sha256:")
		if _, ok := blobs[manifestDigest]; !ok {
			return errors.New("OCI index references a missing manifest")
		}
		manifestData, err := readOCIManifest(file, archiveSize, manifestDigest)
		if err != nil {
			return err
		}
		var image ociManifest
		if json.Unmarshal(manifestData, &image) != nil || image.SchemaVersion != 2 || !validDigest(image.Config.Digest) {
			return errors.New("invalid OCI image manifest")
		}
		descriptors := append([]ociDescriptor{image.Config}, image.Layers...)
		for _, descriptor := range descriptors {
			if !validDigest(descriptor.Digest) {
				return errors.New("invalid OCI blob digest")
			}
			if _, ok := blobs[strings.TrimPrefix(descriptor.Digest, "sha256:")]; !ok {
				return errors.New("OCI image references a missing blob")
			}
		}
	}
	return nil
}

func readOCIManifest(file *os.File, archiveSize int64, digest string) ([]byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	reader := tar.NewReader(io.LimitReader(file, archiveSize))
	for {
		header, err := reader.Next()
		if err != nil {
			return nil, errors.New("OCI index references a missing manifest")
		}
		if header.Name != "blobs/sha256/"+digest {
			continue
		}
		if header.Size > maxOCIMetadataSize {
			return nil, errors.New("OCI manifest exceeds 4 MiB")
		}
		data, err := io.ReadAll(io.LimitReader(reader, maxOCIMetadataSize+1))
		if err != nil || int64(len(data)) != header.Size {
			return nil, errors.New("OCI manifest is truncated")
		}
		return data, nil
	}
}

func scanOCI(file *os.File, archiveSize int64, capture func(string, []byte) error) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	reader := tar.NewReader(io.LimitReader(file, archiveSize))
	seen := make(map[string]struct{})
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid OCI archive: %w", err)
		}
		name := header.Name
		clean := path.Clean(name)
		if strings.Contains(name, "\\") || path.IsAbs(name) || clean != strings.TrimSuffix(name, "/") {
			return errors.New("OCI archive contains an unsafe path")
		}
		if _, ok := seen[name]; ok {
			return errors.New("OCI archive contains duplicate paths")
		}
		seen[name] = struct{}{}
		if header.Typeflag == tar.TypeDir {
			if name != "blobs" && name != "blobs/" && name != "blobs/sha256" && name != "blobs/sha256/" {
				return errors.New("OCI archive contains an unexpected directory")
			}
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return errors.New("OCI archive contains a link or special file")
		}
		if header.Size < 0 || header.Size > archiveSize {
			return errors.New("OCI archive entry size is invalid")
		}
		if name == "oci-layout" || name == "index.json" {
			if header.Size > maxOCIMetadataSize {
				return errors.New("OCI metadata exceeds 4 MiB")
			}
			data, err := io.ReadAll(io.LimitReader(reader, maxOCIMetadataSize+1))
			if err != nil || int64(len(data)) != header.Size {
				return errors.New("OCI metadata is truncated")
			}
			if err := capture(name, data); err != nil {
				return err
			}
			continue
		}
		prefix := "blobs/sha256/"
		if !strings.HasPrefix(name, prefix) || !sha256Digest.MatchString(strings.TrimPrefix(name, prefix)) {
			return errors.New("OCI archive contains an unexpected file")
		}
		hash := sha256.New()
		written, err := io.CopyN(hash, reader, header.Size)
		if err != nil || written != header.Size {
			return errors.New("OCI blob is truncated")
		}
		if hex.EncodeToString(hash.Sum(nil)) != strings.TrimPrefix(name, prefix) {
			return errors.New("OCI blob digest mismatch")
		}
		if err := capture(name, nil); err != nil {
			return err
		}
	}
}

func validDigest(digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	return sha256Digest.MatchString(strings.TrimPrefix(digest, "sha256:"))
}
