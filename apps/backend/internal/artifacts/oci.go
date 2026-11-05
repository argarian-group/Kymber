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
	"strings"
)

const MaxOCIArchiveSize int64 = 1 << 30
const maxOCIMetadataSize int64 = 4 << 20

var sha256Digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

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
