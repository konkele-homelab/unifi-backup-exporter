# UniFi Backup Exporter

A lightweight HTTP service that exposes the most recent UniFi Network backup (`.unf`) from a directory.

The exporter is intended to sit alongside an existing backup process and provide a simple API for downloading the latest backup, retrieving metadata, and performing readiness checks.

The application has no external dependencies and is distributed as a small distroless container image.

---

## Features

- 📁 Serves the newest `.unf` backup file
- 📋 Returns backup metadata as JSON
- 🔒 Calculates SHA256 checksums on demand
- ❤️ Kubernetes-friendly readiness endpoint
- ⚡ Single static Go binary
- 📦 Distroless runtime image

---

## Configuration

The application is configured entirely through environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_DIR` | `/backups` | Directory containing UniFi backup files |
| `LISTEN` | `:8081` | HTTP listen address |

---

## HTTP Endpoints

### `GET /latest`

Returns the newest `.unf` file.

The newest backup is determined by file modification time.

Example:

```text
GET /latest
```

Supports both `GET` and `HEAD`.

Returns:

- `200 OK` — Backup file
- `404 Not Found` — No backups available

---

### `GET /metadata`

Returns metadata about the newest backup.

Example response:

```json
{
  "filename": "autobackup_2026-07-15_02-00.unf",
  "size": 2531840,
  "modified": "2026-07-15T02:00:00Z",
  "sha256": "f2f1d9..."
}
```

Returns:

- `200 OK`
- `404 Not Found` if no backup exists
- `500 Internal Server Error` if the checksum cannot be calculated

---

### `GET /readyz`

Readiness endpoint intended for Kubernetes.

Returns:

- `200 OK` if at least one backup exists
- `503 Service Unavailable` if no backup is available

---

## Running with Docker

```bash
docker run \
  -p 8081:8081 \
  -v /path/to/backups:/backups:ro \
  ghcr.io/your-org/unifi-backup-exporter:latest
```

Or specify a custom directory:

```bash
docker run \
  -e BACKUP_DIR=/data/backups \
  -v /path/to/backups:/data/backups:ro \
  -p 8081:8081 \
  ghcr.io/your-org/unifi-backup-exporter:latest
```

---

## Kubernetes Example

```yaml
containers:
  - name: exporter
    image: ghcr.io/your-org/unifi-backup-exporter:latest
    env:
      - name: BACKUP_DIR
        value: /backups
    ports:
      - containerPort: 8081
    volumeMounts:
      - name: backups
        mountPath: /backups
        readOnly: true
    readinessProbe:
      httpGet:
        path: /readyz
        port: 8081
```

---

## How It Works

On each request the exporter:

1. Scans the configured backup directory.
2. Finds the newest file with a `.unf` extension.
3. Returns either:
   - the file itself (`/latest`),
   - metadata (`/metadata`), or
   - readiness status (`/readyz`).

No metadata database or cache is maintained.

---

## Notes

- Only files ending in `.unf` are considered.
- The latest backup is selected using the file modification timestamp.
- SHA256 hashes are calculated when `/metadata` is requested.
- The exporter is read-only and never modifies backup files.