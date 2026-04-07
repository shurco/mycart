# Migrating from litecart to myCart

This guide covers everything you need to do when upgrading from `litecart` (any version) to `mycart`.

> **Your data is safe.** The database schema, volume paths (`lc_base`, `lc_digitals`, `lc_uploads`, `site`), and configuration files have **not** changed. Only the project name, binary name, Docker image name, Go module path, and repository URL have been updated.

---

## What changed

| Before (litecart) | After (myCart) |
|---|---|
| Binary: `litecart` | Binary: `mycart` |
| Docker image: `shurco/litecart` | Docker image: `shurco/mycart` |
| GHCR image: `ghcr.io/shurco/litecart` | GHCR image: `ghcr.io/shurco/mycart` |
| Homebrew: `brew install shurco/tap/litecart` | Homebrew: `brew install shurco/tap/mycart` |
| Go module: `github.com/shurco/litecart` | Go module: `github.com/shurco/mycart` |
| Repository: `github.com/shurco/litecart` | Repository: `github.com/shurco/mycart` |
| CLI command: `./litecart serve` | CLI command: `./mycart serve` |
| Container name: `litecart` | Container name: `mycart` |

## What did NOT change

- Database format and schema — fully compatible, no migration needed
- Volume mount paths: `./lc_base`, `./lc_digitals`, `./lc_uploads`, `./site`
- API endpoints and response format
- Admin panel URL (`/_/`)
- Configuration stored in the database
- Port defaults (`8080`)

---

## Binary (Linux / macOS / Windows)

### Step 1 — Back up

```bash
cp -r ./lc_base ./lc_base_backup
cp -r ./site ./site_backup
```

### Step 2 — Download the new binary

```bash
curl -L https://raw.githubusercontent.com/shurco/mycart/main/scripts/install | sh
```

Or download manually from [Releases](https://github.com/shurco/mycart/releases/latest).

### Step 3 — Remove the old binary

```bash
rm ./litecart
```

### Step 4 — Run

```bash
./mycart serve
```

The new binary will pick up the existing `./lc_base/data.db` database and all volumes automatically.

---

## Homebrew (macOS)

```bash
brew uninstall litecart
brew untap shurco/tap 2>/dev/null
brew tap shurco/tap
brew install mycart
```

---

## Docker

### Step 1 — Stop the old container

```bash
docker stop litecart
```

### Step 2 — Pull the new image

```bash
# Docker Hub
docker pull shurco/mycart:latest

# or GitHub Container Registry
docker pull ghcr.io/shurco/mycart:latest
```

### Step 3 — Rename the old container (backup)

```bash
docker rename litecart litecart-backup
```

### Step 4 — Start the new container

Use the **same volume mounts** — your data is fully compatible:

```bash
docker run \
  --name mycart \
  --restart unless-stopped \
  -p '8080:8080' \
  -v ./lc_base:/lc_base \
  -v ./lc_digitals:/lc_digitals \
  -v ./lc_uploads:/lc_uploads \
  -v ./site:/site \
  shurco/mycart:latest
```

### Step 5 — Verify and clean up

Once everything works, remove the old container:

```bash
docker rm litecart-backup
docker rmi shurco/litecart:latest
```

---

## Docker Compose

### Step 1 — Update `docker-compose.yml`

Replace the service definition:

```yaml
# Before
services:
  litecart:
    image: shurco/litecart:latest
    container_name: litecart

# After
services:
  mycart:
    image: shurco/mycart:latest
    container_name: mycart
```

### Step 2 — Update nginx config

If you use the bundled nginx reverse proxy, update `docker/nginx/nginx.conf`:

```nginx
# Before
upstream cart {
    server litecart:8080;
}

# After
upstream cart {
    server mycart:8080;
}
```

### Step 3 — Restart

```bash
docker-compose down
docker-compose up -d
```

---

## Kubernetes

### Step 1 — Update manifests

Replace all occurrences in your K8s manifests:

| Resource | Old name | New name |
|---|---|---|
| PVC | `litecart-pvc` | `mycart-pvc` |
| Deployment | `litecart` | `mycart` |
| Container image | `shurco/litecart:latest` | `shurco/mycart:latest` |
| Container name | `litecart` | `mycart` |
| Volume name | `litecart-storage` | `mycart-storage` |
| Service | `litecart-service` | `mycart-service` |
| Ingress | `litecart-ingress` | `mycart-ingress` |
| TLS secret | `litecart-tls` | `mycart-tls` |
| Labels | `app: litecart` | `app: mycart` |

An updated example manifest is available in [`/k8s/`](../k8s/).

### Step 2 — Apply

```bash
kubectl apply -f k8s/mycart.yaml
```

> **Note:** If you are using a PVC with `ReadWriteOnce`, you may need to delete the old deployment first before creating the new one to release the volume.

---

## Go module (for developers importing the package)

If your Go project imports `litecart` packages:

```go
// Before
import "github.com/shurco/litecart/pkg/litepay"

// After
import "github.com/shurco/mycart/pkg/litepay"
```

Update your `go.mod`:

```bash
go get github.com/shurco/mycart@latest
```

Then remove the old dependency:

```bash
go mod tidy
```

---

## CI/CD pipelines

Update any references in your CI/CD configuration:

- Docker image references: `shurco/litecart` → `shurco/mycart`
- Binary download URLs: `github.com/shurco/litecart/releases` → `github.com/shurco/mycart/releases`
- Install script URL: `raw.githubusercontent.com/shurco/litecart/main/scripts/install` → `raw.githubusercontent.com/shurco/mycart/main/scripts/install`
- Go module path: `github.com/shurco/litecart` → `github.com/shurco/mycart`

---

## Troubleshooting

**Q: Will my existing database work with the new version?**
A: Yes. The database schema is unchanged. The new binary reads the same `./lc_base/data.db` file.

**Q: Do I need to re-configure payment systems?**
A: No. All settings are stored in the database and remain intact.

**Q: Will the old repository still be accessible?**
A: The old repository at `github.com/shurco/litecart` will redirect to the new location for a transition period.

**Q: I use the self-update feature (`./litecart update`). Will it work?**
A: No. The old binary looks for releases under the `litecart` repository name. Download `mycart` manually using the instructions above, then future updates via `./mycart update` will work correctly.
