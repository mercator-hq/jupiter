# Systemd Deployment Guide

Run Mercator Jupiter as a systemd service on Linux servers.

## Quick Start

```bash
# Install binary
sudo cp mercator /usr/local/bin/
sudo chmod +x /usr/local/bin/mercator

# Install systemd unit
sudo cp examples/systemd/mercator.service /etc/systemd/system/
sudo cp examples/systemd/mercator.env /etc/mercator/

# Start service
sudo systemctl daemon-reload
sudo systemctl enable mercator
sudo systemctl start mercator
```

---

## Installation

### 1. Install Binary

```bash
# From release
wget https://github.com/mercator-hq/jupiter/releases/download/v0.1.0/mercator-linux-amd64
sudo mv mercator-linux-amd64 /usr/local/bin/mercator
sudo chmod +x /usr/local/bin/mercator

# Verify
mercator version
```

### 2. Create User

```bash
# Create dedicated user
sudo useradd -r -s /bin/false -d /opt/mercator mercator

# Create directories
sudo mkdir -p /opt/mercator/{data,policies}
sudo mkdir -p /etc/mercator
sudo mkdir -p /var/log/mercator

# Set permissions
sudo chown -R mercator:mercator /opt/mercator
sudo chown -R mercator:mercator /var/log/mercator
```

### 3. Configure

```bash
# Configuration
sudo vim /etc/mercator/config.yaml

# Environment variables (API keys)
sudo vim /etc/mercator/mercator.env

# Secure the env file
sudo chmod 600 /etc/mercator/mercator.env
sudo chown root:root /etc/mercator/mercator.env
```

### 4. Create Systemd Unit

```ini
# /etc/systemd/system/mercator.service
[Unit]
Description=Mercator Jupiter - LLM Governance Proxy
Documentation=https://github.com/mercator-hq/jupiter
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mercator
Group=mercator

# Working directory
WorkingDirectory=/opt/mercator

# Binary location
ExecStart=/usr/local/bin/mercator run --config /etc/mercator/config.yaml

# Restart policy
Restart=on-failure
RestartSec=5s

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/mercator /var/log/mercator

# Environment file
EnvironmentFile=/etc/mercator/mercator.env

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mercator

[Install]
WantedBy=multi-user.target
```

### 5. Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable on boot
sudo systemctl enable mercator

# Start service
sudo systemctl start mercator

# Check status
sudo systemctl status mercator
```

---

## Management

### Service Control

```bash
# Start
sudo systemctl start mercator

# Stop
sudo systemctl stop mercator

# Restart
sudo systemctl restart mercator

# Reload config (if supported)
sudo systemctl reload mercator

# Check status
sudo systemctl status mercator

# Enable on boot
sudo systemctl enable mercator

# Disable
sudo systemctl disable mercator
```

### View Logs

```bash
# Follow logs
sudo journalctl -u mercator -f

# Last 100 lines
sudo journalctl -u mercator -n 100

# Today's logs
sudo journalctl -u mercator --since today

# Specific time range
sudo journalctl -u mercator --since "2025-01-01 00:00:00" --until "2025-01-02 00:00:00"

# JSON format
sudo journalctl -u mercator -o json
```

---

## Configuration

### Production Config

```yaml
# /etc/mercator/config.yaml
proxy:
  listen_address: "0.0.0.0:8080"
  read_timeout: "30s"
  write_timeout: "30s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

policy:
  mode: "file"
  file_path: "/opt/mercator/policies/production.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "/opt/mercator/data/evidence.db"
  retention_days: 90

telemetry:
  logging:
    level: "info"
    format: "json"
```

### Environment Variables

```bash
# /etc/mercator/mercator.env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-...

# Optional overrides
MERCATOR_PROXY_LISTEN_ADDRESS=0.0.0.0:8080
MERCATOR_TELEMETRY_LOGGING_LEVEL=info
```

---

## Monitoring

### Health Checks

```bash
# Create health check script
cat > /usr/local/bin/check-mercator-health.sh << 'EOF'
#!/bin/bash
if curl -sf http://localhost:8080/health > /dev/null; then
  exit 0
else
  exit 1
fi
EOF

chmod +x /usr/local/bin/check-mercator-health.sh

# Run from cron
echo "*/5 * * * * /usr/local/bin/check-mercator-health.sh || systemctl restart mercator" | sudo crontab -
```

### Metrics Export

```bash
# Export metrics to file
curl http://localhost:9090/metrics > /tmp/mercator-metrics.txt

# Or use node_exporter for Prometheus
```

---

## Upgrading

```bash
# Stop service
sudo systemctl stop mercator

# Backup
sudo cp /usr/local/bin/mercator /usr/local/bin/mercator.bak
sudo cp -r /opt/mercator/data /opt/mercator/data.bak

# Install new version
wget https://github.com/mercator-hq/jupiter/releases/download/v0.2.0/mercator-linux-amd64
sudo mv mercator-linux-amd64 /usr/local/bin/mercator
sudo chmod +x /usr/local/bin/mercator

# Start service
sudo systemctl start mercator

# Verify
sudo systemctl status mercator
mercator version
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check status
sudo systemctl status mercator

# View errors
sudo journalctl -u mercator -n 50

# Test manually
sudo -u mercator /usr/local/bin/mercator run --config /etc/mercator/config.yaml --dry-run
```

### Permission Issues

```bash
# Fix ownership
sudo chown -R mercator:mercator /opt/mercator
sudo chown -R mercator:mercator /var/log/mercator

# Check SELinux (if enabled)
sudo setsebool -P httpd_can_network_connect 1
```

---

## Best Practices

1. **Run as dedicated user** - Never run as root
2. **Secure environment file** - chmod 600 for secrets
3. **Enable on boot** - systemctl enable
4. **Monitor logs** - journalctl for debugging
5. **Set resource limits** - LimitNOFILE, LimitNPROC
6. **Regular backups** - Backup data directory
7. **Health checks** - Automated monitoring
8. **Log rotation** - journald handles this
9. **Graceful restarts** - Use systemctl reload if supported
10. **Security hardening** - ProtectSystem, NoNewPrivileges

---

## See Also

- [Bare Metal Deployment](bare-metal.md)
- [Docker Deployment](docker.md)
- [High Availability](high-availability.md)
