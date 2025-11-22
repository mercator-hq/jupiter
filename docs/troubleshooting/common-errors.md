# Common Errors & Solutions

Comprehensive troubleshooting guide for common Mercator Jupiter errors.

## Table of Contents

- [Server Startup Errors](#server-startup-errors)
- [Configuration Errors](#configuration-errors)
- [Policy Errors](#policy-errors)
- [Provider Errors](#provider-errors)
- [Evidence Storage Errors](#evidence-storage-errors)
- [Performance Issues](#performance-issues)
- [Network Errors](#network-errors)

---

## Server Startup Errors

### Error: "Configuration file not found"

**Message**: `Error loading configuration: open config.yaml: no such file or directory`

**Cause**: Mercator can't find the configuration file

**Solutions**:
1. Specify config path explicitly:
   ```bash
   mercator run --config /path/to/config.yaml
   ```

2. Create config in default location:
   ```bash
   cp examples/configs/minimal.yaml ./config.yaml
   ```

3. Use absolute path:
   ```bash
   mercator run --config /etc/mercator/config.yaml
   ```

---

### Error: "Port already in use"

**Message**: `Error starting server: listen tcp 127.0.0.1:8080: bind: address already in use`

**Cause**: Another process is using port 8080

**Solutions**:
1. Find and stop the conflicting process:
   ```bash
   # macOS/Linux
   lsof -ti:8080 | xargs kill -9

   # Or check what's using the port
   lsof -i :8080
   ```

2. Change the listen port:
   ```yaml
   proxy:
     listen_address: "127.0.0.1:8081"
   ```

3. Use command-line override:
   ```bash
   mercator run --listen 127.0.0.1:8081
   ```

---

### Error: "Invalid API key"

**Message**: `Provider authentication failed: invalid API key`

**Cause**: API key not set or invalid

**Solutions**:
1. Check environment variable is set:
   ```bash
   echo $OPENAI_API_KEY
   ```

2. Set the API key:
   ```bash
   export OPENAI_API_KEY="sk-..."
   ```

3. Verify key on provider dashboard:
   - OpenAI: https://platform.openai.com/api-keys
   - Anthropic: https://console.anthropic.com/account/keys

4. Check for whitespace:
   ```bash
   # Remove any trailing whitespace
   export OPENAI_API_KEY=$(echo $OPENAI_API_KEY | tr -d '[:space:]')
   ```

---

## Configuration Errors

### Error: "Invalid YAML syntax"

**Message**: `Error parsing config: yaml: line 10: mapping values are not allowed in this context`

**Cause**: YAML syntax error (usually indentation or colons)

**Solutions**:
1. Check indentation (use spaces, not tabs):
   ```yaml
   # Correct
   proxy:
     listen_address: "127.0.0.1:8080"

   # Incorrect (tab instead of spaces)
   proxy:
   	listen_address: "127.0.0.1:8080"
   ```

2. Ensure colons have spaces:
   ```yaml
   # Correct
   key: value

   # Incorrect
   key:value
   ```

3. Validate YAML:
   ```bash
   yamllint config.yaml
   ```

---

### Error: "Missing required field"

**Message**: `Configuration validation failed: providers.openai.api_key is required`

**Cause**: Required configuration field is missing

**Solutions**:
1. Add the missing field:
   ```yaml
   providers:
     openai:
       base_url: "https://api.openai.com/v1"
       api_key: "${OPENAI_API_KEY}"  # Add this
   ```

2. Check all required fields:
   - `proxy.listen_address`
   - `providers.<name>.base_url`
   - `providers.<name>.api_key`
   - `policy.file_path` or `policy.git_repo`

3. Use dry-run to validate:
   ```bash
   mercator run --config config.yaml --dry-run
   ```

---

### Error: "Invalid duration format"

**Message**: `Configuration validation failed: timeout: invalid duration "60"`

**Cause**: Duration missing time unit

**Solutions**:
```yaml
# Incorrect
timeout: 60

# Correct
timeout: "60s"  # Must include unit: s, m, h
```

**Valid units**: `s` (seconds), `m` (minutes), `h` (hours)

---

## Policy Errors

### Error: "Policy file not found"

**Message**: `Error loading policies: open policies.yaml: no such file or directory`

**Cause**: Policy file doesn't exist at specified path

**Solutions**:
1. Create a basic policy file:
   ```bash
   cat > policies.yaml << 'EOF'
   version: "1.0"
   policies:
     - name: "allow-all"
       rules:
         - condition: "true"
           action: "allow"
   EOF
   ```

2. Check the path in config:
   ```yaml
   policy:
     mode: "file"
     file_path: "./policies.yaml"  # Verify this path
   ```

3. Use absolute path:
   ```yaml
   policy:
     file_path: "/etc/mercator/policies.yaml"
   ```

---

### Error: "Policy validation failed"

**Message**: `Policy validation failed: policies[0].rules[0].condition: syntax error`

**Cause**: Invalid MPL syntax in policy

**Solutions**:
1. Lint the policy:
   ```bash
   mercator lint --file policies.yaml
   ```

2. Check condition syntax:
   ```yaml
   # Incorrect
   condition: request.model = "gpt-4"

   # Correct
   condition: 'request.model == "gpt-4"'
   ```

3. Common syntax errors:
   - Use `==` for equality, not `=`
   - Quote string values: `"gpt-4"` not `gpt-4`
   - Use `and`/`or` for boolean logic, not `&&`/`||`

See: [MPL Syntax Guide](../mpl/SYNTAX.md)

---

### Error: "Unknown action type"

**Message**: `Policy validation failed: unknown action: "block"`

**Cause**: Invalid action name (should be "deny" not "block")

**Solutions**:
```yaml
# Incorrect
action: "block"

# Correct
action: "deny"
```

**Valid actions**: `allow`, `deny`, `log`, `route`, `redact`, `modify`

---

## Provider Errors

### Error: "Provider not responding"

**Message**: `Provider error: dial tcp: i/o timeout`

**Cause**: Can't connect to provider API

**Solutions**:
1. Check network connectivity:
   ```bash
   ping api.openai.com
   curl https://api.openai.com/v1/models -I
   ```

2. Check firewall rules:
   - Allow outbound HTTPS (port 443)
   - Check corporate proxy settings

3. Verify provider status:
   - OpenAI: https://status.openai.com/
   - Anthropic: https://status.anthropic.com/

4. Increase timeout:
   ```yaml
   providers:
     openai:
       timeout: "120s"  # Increase from default
   ```

---

### Error: "Rate limit exceeded"

**Message**: `Provider error: rate limit exceeded (429)`

**Cause**: Too many requests to provider

**Solutions**:
1. Check your provider tier limits
2. Implement rate limiting in Jupiter:
   ```yaml
   limits:
     rate_limiting:
       enabled: true
       default_rpm: 50  # Below your provider limit
   ```

3. Use multiple API keys:
   ```yaml
   providers:
     openai-1:
       api_key: "${OPENAI_API_KEY_1}"
     openai-2:
       api_key: "${OPENAI_API_KEY_2}"
   ```

4. Add retry logic with backoff

---

### Error: "Model not found"

**Message**: `Provider error: The model 'gpt-5' does not exist`

**Cause**: Invalid model name or model not available

**Solutions**:
1. Check model name spelling:
   ```yaml
   # Incorrect
   model: "gpt-3.5"

   # Correct
   model: "gpt-3.5-turbo"
   ```

2. Verify model availability:
   ```bash
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"
   ```

3. Check model access:
   - Some models require special access
   - Verify your API key has access

---

## Evidence Storage Errors

### Error: "Database locked"

**Message**: `Evidence storage error: database is locked`

**Cause**: Multiple processes trying to access SQLite database

**Solutions**:
1. Ensure only one Jupiter instance per database:
   ```bash
   # Check for multiple instances
   pgrep mercator
   ```

2. Use separate databases for multiple instances:
   ```yaml
   evidence:
     sqlite:
       path: "./evidence-instance-1.db"
   ```

3. Enable WAL mode (enabled by default):
   ```yaml
   evidence:
     sqlite:
       wal_mode: true
   ```

4. For production, use PostgreSQL:
   ```yaml
   evidence:
     backend: "postgres"
   ```

---

### Error: "Disk full"

**Message**: `Evidence storage error: no space left on device`

**Cause**: Disk full from evidence storage

**Solutions**:
1. Check disk space:
   ```bash
   df -h
   ```

2. Enable retention pruning:
   ```yaml
   evidence:
     retention_days: 30  # Reduce from 90
   ```

3. Manually prune old evidence:
   ```bash
   # Delete evidence older than 30 days
   sqlite3 evidence.db "DELETE FROM evidence WHERE timestamp < datetime('now', '-30 days');"
   ```

4. Move database to larger disk:
   ```yaml
   evidence:
     sqlite:
       path: "/data/evidence.db"
   ```

---

## Performance Issues

### Issue: "High memory usage"

**Symptoms**: Memory usage growing over time

**Causes & Solutions**:

1. **Large policy files**: Optimize policies
2. **Evidence buffer full**: Reduce buffer size:
   ```yaml
   evidence:
     recorder:
       async_buffer: 500  # Reduce from 1000
   ```

3. **Connection pool leaks**: Restart server
4. **Check for memory leaks**:
   ```bash
   # Monitor memory
   top -pid $(pgrep mercator)
   ```

---

### Issue: "High latency"

**Symptoms**: Slow response times

**Causes & Solutions**:

1. **Slow provider**: Check provider latency
   ```bash
   mercator benchmark --target http://localhost:8080
   ```

2. **Complex policies**: Optimize policy conditions
3. **Network latency**: Use closer provider region
4. **Resource limits**: Increase CPU/memory
5. **Database contention**: Use PostgreSQL instead of SQLite

---

### Issue: "High CPU usage"

**Symptoms**: CPU at 100%

**Causes & Solutions**:

1. **Policy evaluation**: Simplify complex regex patterns
2. **Logging overhead**: Reduce log level:
   ```yaml
   telemetry:
     logging:
       level: "warn"  # Reduce from "debug"
   ```

3. **Many concurrent requests**: Scale horizontally
4. **Check for loops**:
   ```bash
   # Check CPU usage
   top -o cpu | grep mercator
   ```

---

## Network Errors

### Error: "Connection refused"

**Message**: `dial tcp 127.0.0.1:8080: connect: connection refused`

**Cause**: Mercator server not running

**Solutions**:
1. Start the server:
   ```bash
   mercator run --config config.yaml
   ```

2. Check server is listening:
   ```bash
   lsof -i :8080
   netstat -an | grep 8080
   ```

3. Check server logs for startup errors

---

### Error: "TLS handshake failed"

**Message**: `x509: certificate signed by unknown authority`

**Cause**: TLS certificate issues

**Solutions**:
1. Check certificate is valid:
   ```bash
   openssl x509 -in cert.pem -text -noout
   ```

2. Verify certificate chain is complete
3. Check certificate file path in config:
   ```yaml
   security:
     tls:
       cert_file: "/etc/mercator/tls/cert.pem"  # Verify path
       key_file: "/etc/mercator/tls/key.pem"
   ```

4. For development, disable TLS verification (not recommended for production)

---

## Debug Mode

Enable debug logging to troubleshoot issues:

```yaml
telemetry:
  logging:
    level: "debug"
    format: "text"  # More readable than JSON
```

Or via command line:
```bash
mercator run --config config.yaml --log-level debug
```

## Getting Help

If you're still stuck:

1. **Check logs**:
   ```bash
   mercator run --config config.yaml --log-level debug 2>&1 | tee debug.log
   ```

2. **Validate configuration**:
   ```bash
   mercator run --config config.yaml --dry-run
   ```

3. **Validate policies**:
   ```bash
   mercator lint --file policies.yaml --strict
   ```

4. **Check provider health**:
   ```bash
   curl http://localhost:8080/health
   ```

5. **Review metrics**:
   ```bash
   curl http://localhost:8080/metrics
   ```

6. **Search documentation**: [docs/](../)

7. **File an issue**: https://github.com/mercator-hq/jupiter/issues

---

## See Also

- [Provider Issues](provider-issues.md) - Provider-specific errors
- [Policy Errors](policy-errors.md) - Policy validation issues
- [Performance Tuning](performance.md) - Optimization guide
- [Debugging Guide](debugging.md) - Advanced debugging

---

## Quick Troubleshooting Checklist

```bash
# 1. Check configuration is valid
mercator run --config config.yaml --dry-run

# 2. Validate policies
mercator lint --file policies.yaml

# 3. Check server is running
ps aux | grep mercator

# 4. Check server is listening
lsof -i :8080

# 5. Check provider connectivity
curl https://api.openai.com/v1/models -I

# 6. Check environment variables
echo $OPENAI_API_KEY

# 7. Check disk space
df -h

# 8. Check server health
curl http://localhost:8080/health

# 9. Check server logs
mercator run --log-level debug

# 10. Check metrics
curl http://localhost:8080/metrics | grep error
```
