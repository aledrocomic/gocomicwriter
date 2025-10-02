# Deploying gcwserver to AWS EC2 (Debian) with GoLand 2025.2

This how-to walks you through deploying the Go Comic Writer backend server (gcwserver) on an AWS EC2 Debian instance. It assumes:
- Your EC2 instance can reach an Amazon RDS PostgreSQL instance (via Security Group rules).  
- You have SSH access to the instance.  
- You use GoLand 2025.2 and may want to run/debug on the EC2 host via SSH Targets or JetBrains Remote Development.

The steps cover a production-friendly baseline using systemd and an optional Nginx reverse proxy + Let’s Encrypt TLS. You can also terminate TLS on an AWS Load Balancer instead of Nginx.

---

## 1) Prerequisites

- AWS
  - EC2: Debian 12 (Bookworm) or Debian 13 (Trixie), t3.small+ recommended. Assign a public IP or access via a bastion/SSM.  
  - Security Groups:  
    - Inbound: allow 22/tcp from your IP for SSH.  
    - If exposing directly: allow 80/tcp (HTTP) and 443/tcp (HTTPS) from the Internet or your allowed CIDRs.  
    - Outbound: allow to the RDS security group / RDS endpoint on 5432/tcp.  
  - RDS PostgreSQL: note the endpoint, database name, username, and password. Ensure SSL is enabled (default). Add the EC2 instance’s security group to the RDS inbound rules if using SG-to-SG.
- Local machine: GoLand 2025.2 installed
- DNS: an A/AAAA record to your EC2 public IP (if you’ll use Nginx + Let’s Encrypt)

---

## 2) Prepare the EC2 Debian instance

SSH to the instance and install base packages.

```
sudo apt update
sudo apt -y upgrade
sudo apt -y install ca-certificates curl unzip
```

Create a dedicated system user and directories for gcwserver:

```
sudo useradd --system --home /opt/gcwserver --shell /usr/sbin/nologin gcw
sudo mkdir -p /opt/gcwserver/bin /opt/gcwserver/var /opt/gcwserver/log
sudo chown -R gcw:gcw /opt/gcwserver
```

---

## 3) Obtain gcwserver

You have two options.

- Option A — Use a prebuilt release binary (recommended for quick deploy):
  - Download the Linux binary for your CPU from the project’s Releases. For example (amd64):
    ```
    # Replace URL with the actual release URL for your version
    curl -L -o /tmp/gcwserver https://example.com/releases/gcwserver_linux_amd64
    sudo install -m 0755 /tmp/gcwserver /opt/gcwserver/bin/gcwserver
    ```

- Option B — Build on the instance from source (useful for GoLand remote run/debug):
  - Install Go (if not already):
    ```
    sudo apt -y install golang
    go version
    ```
  - Clone the repository:
    ```
    sudo -u gcw -H bash -lc "cd /opt && git clone https://github.com/your-org/gocomicwriter.git"
    sudo ln -s /opt/gocomicwriter/dist /opt/gcwserver/dist 2>/dev/null || true
    ```
  - Build:
    ```
    sudo -u gcw -H bash -lc "cd /opt/gocomicwriter && go build -o /opt/gcwserver/bin/gcwserver ./cmd/gcwserver"
    ```

Note: This repo also contains sample prebuilt binaries under dist/ for various platforms; for production, prefer official signed release artifacts.

---

## 4) Configure environment (RDS, auth, ports, TLS)

gcwserver reads configuration from environment variables. Create an env file:

```
sudo tee /etc/gcwserver.env > /dev/null <<'EOF'
# PostgreSQL (use either GCW_PG_DSN or DATABASE_URL)
# Example for RDS with SSL required:
GCW_PG_DSN=postgres://gcwuser:StrongPassword@your-rds-endpoint:5432/gcwdb?sslmode=require

# Bind address
# If using Nginx/ALB in front, bind to localhost only
ADDR=127.0.0.1:8080
# Alternatively, use PORT=8080 to imply :8080

# Authentication
# dev: convenient for local testing; static: production-style with admin API key gated token issuance
GCW_AUTH_MODE=static
# Long, random string used to sign tokens issued by /api/auth/token
GCW_AUTH_SECRET=$(openssl rand -hex 32)
# Required if GCW_AUTH_MODE=static; passed via X-API-Key on token creation
GCW_ADMIN_API_KEY=$(openssl rand -hex 32)

# TLS (leave disabled when behind Nginx/ALB)
GCW_TLS_ENABLE=false
GCW_TLS_CERT_FILE=
GCW_TLS_KEY_FILE=

# Object store health (optional). If you have a MinIO/S3 gateway or service to check, set this.
# GCW_OBJECT_HEALTH_URL=http://minio.internal:9000/minio/health/ready
# If required=true, /readyz fails when object health is down.
GCW_OBJECT_HEALTH_REQUIRED=false
EOF
```

Security: replace the placeholder values with real credentials. If you paste secrets directly (instead of command substitution), remove the `$()` around openssl.

Reference: if GCW_PG_DSN is not set, gcwserver also checks `DATABASE_URL`. Default (dev) fallback is a local Postgres URL.

---

## 5) Create a systemd service

```
sudo tee /etc/systemd/system/gcwserver.service > /dev/null <<'EOF'
[Unit]
Description=Go Comic Writer Server (gcwserver)
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=gcw
Group=gcw
EnvironmentFile=/etc/gcwserver.env
WorkingDirectory=/opt/gcwserver
ExecStart=/opt/gcwserver/bin/gcwserver
Restart=on-failure
RestartSec=5s
# Hardening (tweak as needed)
NoNewPrivileges=yes
ProtectSystem=full
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now gcwserver
sudo systemctl status gcwserver --no-pager
```

Check health endpoints:

```
curl -s http://127.0.0.1:8080/healthz | jq .
curl -s http://127.0.0.1:8080/readyz | jq .
```

On startup, gcwserver pings the DB and applies embedded migrations automatically.

---

## 6) Add Nginx reverse proxy + Let’s Encrypt (optional)

If you want public HTTPS without an ALB, install Nginx and Certbot.

```
sudo apt -y install nginx
# Open firewall if using ufw (Debian usually doesn’t enable ufw by default)
# sudo ufw allow 'Nginx Full'
```

Basic Nginx site (replace example.com):

```
sudo tee /etc/nginx/sites-available/gcwserver.conf > /dev/null <<'EOF'
server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/gcwserver.conf /etc/nginx/sites-enabled/gcwserver.conf
sudo nginx -t && sudo systemctl reload nginx
```

Issue a certificate with Certbot (via Snap is typical on Debian):

```
sudo apt -y install snapd
sudo snap install core; sudo snap refresh core
sudo snap install --classic certbot
sudo ln -s /snap/bin/certbot /usr/bin/certbot
sudo certbot --nginx -d example.com
```

Nginx will be updated to listen on 443 with your certificate. Keep gcwserver bound to 127.0.0.1:8080.

Alternative: terminate TLS on an AWS Application Load Balancer (ALB) and forward HTTP to the instance.

---

## 7) Issue a token and call an API (sanity test)

With GCW_AUTH_MODE=static, you must provide the admin API key to mint a token.

```
# Create a short-lived token (1h) for a user subject (email recommended)
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $GCW_ADMIN_API_KEY" \
  -d '{"email":"user@example.com","display_name":"User","ttl_seconds":3600}' \
  http://example.com/api/auth/token | jq .

# Use the token to call a protected endpoint
TOKEN="<paste token>"
curl -s -H "Authorization: Bearer $TOKEN" http://example.com/api/projects | jq .
```

Health endpoints (no auth): `/healthz`, `/readyz`, `/version`.

---

## 8) GoLand 2025.2 workflows on EC2

You can use GoLand to run or debug gcwserver directly on the EC2 host. Two common approaches are supported.

### A) SSH Targets: Run/Debug on remote host

1. In GoLand, open your local project (same repo).
2. Configure an SSH Target:  
   - File → Settings → Build, Execution, Deployment → SSH Targets → +  
   - Host: your EC2 public IP or DNS, User: admin user (e.g., `admin`), Auth: private key.  
   - Test connection.
3. Create a Run/Debug Configuration for gcwserver:  
   - Run → Edit Configurations → + → Go Build (or Go Application).  
   - Package/Directory: `cmd/gcwserver`  
   - Run on: select your SSH Target.  
   - Working directory (remote): a directory on EC2 where the code will be synced (e.g., `/home/admin/gocomicwriter`). GoLand will upload sources automatically.  
   - Environment: add the same variables you used in `/etc/gcwserver.env` (e.g., `GCW_PG_DSN`, `GCW_AUTH_MODE`, `GCW_AUTH_SECRET`, `GCW_ADMIN_API_KEY`, `ADDR=127.0.0.1:8080`).  
   - Go toolchain: GoLand can install a remote Go SDK if not present. Allow it when prompted.
4. Stop the systemd service to free the port while you run from GoLand:
   ```
   sudo systemctl stop gcwserver
   ```
5. Click Run or Debug in GoLand. The app will build and run on the EC2 host. Use the Debugger to set breakpoints, inspect variables, etc.
6. When finished, stop the Run/Debug session and re-start the systemd service:
   ```
   sudo systemctl start gcwserver
   ```

Tips:
- If you’re behind Nginx, keep `ADDR=127.0.0.1:8080` so inbound traffic still flows via the proxy.  
- For testing from your machine, use an SSH tunnel (e.g., `ssh -L 8080:127.0.0.1:8080 ec2-user@host`).

### B) JetBrains Remote Development (Gateway)

1. Install JetBrains Gateway locally.  
2. Launch Gateway → New Remote Session → SSH → connect to the EC2 host.  
3. Choose GoLand 2025.2 as the IDE backend (Gateway will install it remotely).  
4. Open the gocomicwriter project on the remote host.  
5. Configure a Run/Debug configuration for `cmd/gcwserver` inside the remote IDE, add environment variables (as above), and run or debug.

This approach runs the full IDE backend on the EC2 instance, reducing file sync overhead on large projects.

---

## 9) Operations

- Logs (systemd):
  ```
  sudo journalctl -u gcwserver -f
  ```
- Restart after updating the binary:
  ```
  sudo systemctl restart gcwserver
  ```
- Zero-downtime with ALB: deploy a new ASG/instance and rotate; or run two instances behind an ALB Target Group and drain connections.
- Security:
  - Prefer `GCW_AUTH_MODE=static` in production and keep `GCW_ADMIN_API_KEY` secret.  
  - Keep `GCW_AUTH_SECRET` long and random; rotate periodically.  
  - Bind to localhost and use Nginx/ALB for public TLS.  
  - Scope the RDS user with least privileges.

---

## Appendix: Environment variables (server)

The server reads these environment variables:

- Database
  - `GCW_PG_DSN` — PostgreSQL DSN. Example: `postgres://user:pass@rds-endpoint:5432/dbname?sslmode=require`  
  - `DATABASE_URL` — alternative var name if `GCW_PG_DSN` is not set.  
- Network
  - `ADDR` — listen address, e.g., `:8080` or `127.0.0.1:8080` (default `:8080`).  
  - `PORT` — alternative to set the port only (implies `ADDR=:$PORT`).
- TLS (for direct TLS termination by gcwserver; optional)
  - `GCW_TLS_ENABLE` — `true|false` (default `false`).  
  - `GCW_TLS_CERT_FILE` — path to certificate (PEM).  
  - `GCW_TLS_KEY_FILE` — path to private key (PEM).  
- Auth
  - `GCW_AUTH_MODE` — `dev|static` (default `dev`). Use `static` in production.  
  - `GCW_ADMIN_API_KEY` — required header `X-API-Key` for `/api/auth/token` when `static` mode is enabled.  
  - `GCW_AUTH_SECRET` — token signing secret. If omitted, a weak dev secret is used and a warning is logged.  
- Health checks for object store (optional)
  - `GCW_OBJECT_HEALTH_URL` — URL checked by `/readyz`, e.g., `http://minio:9000/minio/health/ready`.  
  - `GCW_MINIO_ENDPOINT` — alternative from which the health URL is derived.  
  - `GCW_OBJECT_HEALTH_REQUIRED` — `true|false` If true, `/readyz` fails when object health check fails.

Notes:
- On startup, gcwserver pings the DB and applies schema migrations automatically from embedded SQL files.  
- Health endpoints: `/healthz`, `/readyz`, `/version`.  
- Auth: obtain a token from `/api/auth/token` (POST). Use as `Authorization: Bearer <token>` for protected endpoints like `/api/projects`.  
- Default DB (dev) when not configured: `postgres://postgres:postgres@localhost:5432/gocomicwriter?sslmode=disable`.

---

## Troubleshooting

- `ping db: ...` on startup: verify `GCW_PG_DSN`/`DATABASE_URL`, RDS SG rules, and that `sslmode=require` is present for RDS.  
- 503 from `/readyz`: check DB reachability or your object health URL; set `GCW_OBJECT_HEALTH_REQUIRED=false` to avoid hard failures.  
- 401 on `/api/auth/token` in `static` mode: ensure you pass `-H "X-API-Key: $GCW_ADMIN_API_KEY"`.  
- 401 on protected endpoints: ensure `Authorization: Bearer <token>` is present and not expired.  
- Port already in use when debugging: stop the systemd service before running from GoLand, then start it again afterward.
