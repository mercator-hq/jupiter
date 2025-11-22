# Bare Metal Deployment Guide

Deploy Mercator Jupiter directly on physical or virtual servers.

## Overview

This guide covers deploying Jupiter on bare metal servers without containerization or orchestration.

---

## Prerequisites

- Linux server (Ubuntu 20.04+, RHEL 8+, or similar)
- Go 1.21+ (for building from source)
- SQLite 3.x (for evidence storage)
- Root or sudo access
- Open ports: 8080 (API), 9090 (metrics)

---

## Installation

### Option 1: Pre-built Binary

```bash
# Download latest release
wget https://github.com/mercator-hq/jupiter/releases/download/v0.1.0/mercator-linux-amd64
chmod +x mercator-linux-amd64
sudo mv mercator-linux-amd64 /usr/local/bin/mercator

# Verify installation
mercator version
```

### Option 2: Build from Source

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y git build-essential sqlite3

# Install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build
git clone https://github.com/mercator-hq/jupiter.git
cd jupiter
go build -o mercator ./cmd/mercator

# Install
sudo mv mercator /usr/local/bin/
```

---

## Configuration

### 1. Create Directory Structure

```bash
# Create directories
sudo mkdir -p /opt/mercator/{config,policies,data,logs}
sudo mkdir -p /var/log/mercator

# Create user
sudo useradd -r -s /bin/false -d /opt/mercator mercator

# Set ownership
sudo chown -R mercator:mercator /opt/mercator
sudo chown -R mercator:mercator /var/log/mercator
```

### 2. Create Configuration

```yaml
# /opt/mercator/config/config.yaml
proxy:
  listen_address: "0.0.0.0:8080"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

policy:
  mode: "file"
  file_path: "/opt/mercator/policies/policy.yaml"
  watch: true  # Auto-reload on changes

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "/opt/mercator/data/evidence.db"
  retention_days: 90
  prune_interval: "24h"

routing:
  strategy: "round-robin"
  health_check_interval: "30s"

limits:
  budgets:
    enabled: true
    default_daily_limit: 100.0
  rate_limiting:
    enabled: true
    default_rpm: 60

telemetry:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: true
    prometheus_path: "/metrics"
```

### 3. Set Environment Variables

```bash
# Create environment file
cat > /opt/mercator/config/.env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-...
EOF

# Secure it
sudo chmod 600 /opt/mercator/config/.env
sudo chown mercator:mercator /opt/mercator/config/.env
```

---

## Running

### Interactive Mode (Testing)

```bash
# Load environment
export $(cat /opt/mercator/config/.env | xargs)

# Run
mercator run --config /opt/mercator/config/config.yaml
```

### Background Mode

```bash
# Using nohup
nohup mercator run --config /opt/mercator/config/config.yaml \
  > /var/log/mercator/output.log 2>&1 &

# Get PID
echo $! > /var/run/mercator.pid
```

### With Systemd (Recommended)

See [Systemd Deployment Guide](systemd.md) for proper service management.

---

## Process Management

### Using `supervisord`

```ini
# /etc/supervisor/conf.d/mercator.conf
[program:mercator]
command=/usr/local/bin/mercator run --config /opt/mercator/config/config.yaml
directory=/opt/mercator
user=mercator
autostart=true
autorestart=true
startretries=3
redirect_stderr=true
stdout_logfile=/var/log/mercator/supervisor.log
environment=OPENAI_API_KEY="%(ENV_OPENAI_API_KEY)s",ANTHROPIC_API_KEY="%(ENV_ANTHROPIC_API_KEY)s"
```

```bash
# Start with supervisor
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start mercator
sudo supervisorctl status mercator
```

---

## Networking

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow 8080/tcp comment 'Mercator API'
sudo ufw allow 9090/tcp comment 'Mercator Metrics'

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 9090 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4

# firewalld (RHEL/CentOS)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=9090/tcp
sudo firewall-cmd --reload
```

### Reverse Proxy (Nginx)

```nginx
# /etc/nginx/sites-available/mercator
upstream mercator_backend {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name jupiter.example.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name jupiter.example.com;

    ssl_certificate /etc/ssl/certs/jupiter.crt;
    ssl_certificate_key /etc/ssl/private/jupiter.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://mercator_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # For streaming
        proxy_buffering off;
        proxy_read_timeout 300s;
    }

    location /metrics {
        deny all;  # Protect metrics endpoint
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/mercator /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## Monitoring

### Health Checks

```bash
# Create health check script
cat > /usr/local/bin/check-mercator.sh << 'EOF'
#!/bin/bash
if curl -sf http://localhost:8080/health > /dev/null; then
  echo "Mercator is healthy"
  exit 0
else
  echo "Mercator is unhealthy"
  exit 1
fi
EOF

chmod +x /usr/local/bin/check-mercator.sh

# Add to cron (every 5 minutes)
echo "*/5 * * * * /usr/local/bin/check-mercator.sh || systemctl restart mercator" | crontab -
```

### Log Rotation

```bash
# /etc/logrotate.d/mercator
/var/log/mercator/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0644 mercator mercator
    postrotate
        systemctl reload mercator > /dev/null 2>&1 || true
    endscript
}
```

---

## Backup

### Evidence Data

```bash
# Create backup script
cat > /usr/local/bin/backup-mercator.sh << 'EOF'
#!/bin/bash
BACKUP_DIR=/var/backups/mercator
DATE=$(date +%Y%m%d)

mkdir -p $BACKUP_DIR
tar czf $BACKUP_DIR/mercator-$DATE.tar.gz \
  /opt/mercator/data \
  /opt/mercator/config \
  /opt/mercator/policies

# Keep last 30 days
find $BACKUP_DIR -name "mercator-*.tar.gz" -mtime +30 -delete
EOF

chmod +x /usr/local/bin/backup-mercator.sh

# Run daily at 2 AM
echo "0 2 * * * /usr/local/bin/backup-mercator.sh" | crontab -
```

---

## Troubleshooting

### Check Process

```bash
# Is it running?
ps aux | grep mercator

# Check port
netstat -tlnp | grep 8080
# or
ss -tlnp | grep 8080
```

### Test Configuration

```bash
mercator run --config /opt/mercator/config/config.yaml --dry-run
```

### View Logs

```bash
# If using systemd
journalctl -u mercator -f

# If using nohup
tail -f /var/log/mercator/output.log

# If using supervisor
tail -f /var/log/mercator/supervisor.log
```

---

## Best Practices

1. **Use systemd** - Better process management
2. **Run as dedicated user** - Never run as root
3. **Enable firewall** - Restrict access to necessary ports
4. **Use reverse proxy** - Nginx or HAProxy for TLS
5. **Regular backups** - Automated daily backups
6. **Log rotation** - Prevent disk space issues
7. **Health monitoring** - Automated health checks
8. **Resource limits** - ulimit for file descriptors
9. **Security updates** - Keep system packages updated
10. **Documentation** - Document your specific setup

---

## See Also

- [Systemd Deployment](systemd.md) - Recommended for production
- [Docker Deployment](docker.md) - Containerized deployment
- [High Availability](high-availability.md) - Multi-server setup
