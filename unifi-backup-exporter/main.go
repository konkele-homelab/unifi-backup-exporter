package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type BackupMetadata struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	SHA256   string    `json:"sha256"`
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func newestBackup(dir string) (string, os.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}

	var newestInfo os.FileInfo
	var newestPath string

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".unf" {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if newestInfo == nil || info.ModTime().After(newestInfo.ModTime()) {
			newestInfo = info
			newestPath = filepath.Join(dir, e.Name())
		}
	}

	if newestInfo == nil {
		return "", nil, os.ErrNotExist
	}

	return newestPath, newestInfo, nil
}

func mustDir() string {
	if v := os.Getenv("BACKUP_DIR"); v != "" {
		return v
	}
	return "/backups"
}

func handleReady(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := newestBackup(dir); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func handleMetadata(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path, info, err := newestBackup(dir)
		if err != nil {
			http.Error(w, "no backups", http.StatusNotFound)
			return
		}

		sum, err := fileSHA256(path)
		if err != nil {
			http.Error(w, "hash error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(BackupMetadata{
			Filename: filepath.Base(path),
			Size:     info.Size(),
			Modified: info.ModTime().UTC(),
			SHA256:   sum,
		})
	}
}

func handleLatest(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path, _, err := newestBackup(dir)
		if err != nil {
			http.Error(w, "no backups", http.StatusNotFound)
			return
		}

		// optional safety check (prevents edge-case race conditions)
		if _, err := os.Stat(path); err != nil {
			http.Error(w, "missing file", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, path)
	}
}

func main() {
	dir := mustDir()
	addr := os.Getenv("LISTEN")
	if addr == "" {
		addr = ":8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", handleReady(dir))
	mux.HandleFunc("/metadata", handleMetadata(dir))
	mux.HandleFunc("/latest", handleLatest(dir))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	_ = srv.ListenAndServe()
}