# Policy Manager Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Mercator Jupiter Policy Manager.

## Table of Contents

1. [Policy Loading Issues](#policy-loading-issues)
2. [Hot-Reload Problems](#hot-reload-problems)
3. [Validation Errors](#validation-errors)
4. [Performance Issues](#performance-issues)
5. [File System Issues](#file-system-issues)
6. [Configuration Problems](#configuration-problems)
7. [Debugging Tips](#debugging-tips)

---

## Policy Loading Issues

### Symptom: Policy files not loading

**Check list:**

```bash
# 1. Verify file exists and is readable
ls -la /path/to/policy.yaml

# 2. Check file permissions
# Files must be readable by the process user
chmod 644 /path/to/policy.yaml

# 3. Verify YAML syntax
# Use a YAML validator or:
python3 -c "import yaml; yaml.safe_load(open('/path/to/policy.yaml'))"

# 4. Check file size (must be < 10MB)
ls -lh /path/to/policy.yaml
```

**Common causes:**

- **Permission denied**: File not readable by process user
  - Solution: `chmod 644 policy.yaml` or adjust ownership

- **File not found**: Incorrect path in configuration
  - Solution: Use absolute paths or verify relative path is correct

- **Invalid YAML syntax**: Malformed YAML
  - Solution: Validate YAML syntax, check for tabs vs spaces, unmatched quotes

- **File too large**: Exceeds 10MB limit
  - Solution: Split into multiple smaller files or increase MaxFileSize in LoaderConfig

**Log messages to look for:**

```
ERROR Failed to load policies error="failed to access policy path: ..."
ERROR Failed to load policies error="parse error in ..."
```

### Symptom: Some policies load, others don't (directory mode)

**Check list:**

```bash
# 1. List all YAML files in directory
find /path/to/policies -name "*.yaml" -o -name "*.yml"

# 2. Check for hidden files (skipped by default)
ls -la /path/to/policies/

# 3. Verify each file individually
for f in /path/to/policies/*.yaml; do
    echo "Checking $f"
    python3 -c "import yaml; yaml.safe_load(open('$f'))" || echo "INVALID: $f"
done
```

**Common causes:**

- **Mixed valid/invalid files**: One invalid file blocks all loading (atomic behavior)
  - Solution: Fix or remove the invalid file

- **Hidden files**: Files starting with `.` are skipped by default
  - Solution: Rename files or adjust SkipHidden configuration

- **Wrong extension**: Only `.yaml` and `.yml` files loaded
  - Solution: Rename files with correct extension

**Atomic loading behavior:**

The policy manager uses atomic loading - either **all** policies load or **none** do. This prevents partial policy sets from being active.

```
INFO Loading policies from directory
ERROR Failed to load policies - keeping previous policies
```

If you see this pattern, one file is causing all to fail. Enable debug logging to identify which file.

---

## Hot-Reload Problems

### Symptom: File changes not triggering reload

**Check list:**

```bash
# 1. Verify watch is enabled
grep "watch:" config.yaml

# 2. Check logs for watcher status
# Should see: "File watcher started"

# 3. Test file system events
# On macOS/Linux:
fswatch -1 /path/to/policy.yaml &
echo "test" >> /path/to/policy.yaml

# 4. Verify file is actually changing
stat /path/to/policy.yaml
```

**Common causes:**

- **Watch disabled**: `watch: false` in configuration
  - Solution: Set `watch: true` in policy config

- **Network filesystem**: fsnotify doesn't work on NFS/CIFS
  - Solution: Use local filesystem or implement polling

- **Atomic writes**: Some editors use atomic file replacement
  - Solution: This is handled by the watcher, but ensure proper permissions

- **Too many files**: inotify limits on Linux
  - Solution: Increase `fs.inotify.max_user_watches` system limit

**Log messages:**

```
INFO File watcher started path=/path/to/policies
DEBUG File event detected path=/path/to/policy.yaml op=WRITE
INFO Triggering policy reload
```

### Symptom: Reload happening too frequently

**Cause:** Multiple file events firing in quick succession

**Solution:** Debouncing is built-in (100ms default). If still seeing issues:

```go
// Adjust debounce period in FileWatcherConfig
watchConfig := &FileWatcherConfig{
    Path: policyPath,
    DebounceDuration: 500 * time.Millisecond, // Increase from 100ms
}
```

### Symptom: Changes detected but old policy still active

**Check list:**

```bash
# 1. Check for validation errors
# Look for: "Policy validation failed during reload, keeping previous policies"

# 2. Verify new policy is valid
mercator lint /path/to/policy.yaml

# 3. Check policy version matches
# If version doesn't change, cache might be involved
```

**Common causes:**

- **Validation failure**: New policy fails validation, old kept
  - Solution: Fix validation errors, check with `mercator lint`

- **Parsing error**: New file has syntax errors
  - Solution: Validate YAML syntax

- **Version unchanged**: Some systems cache based on content hash
  - Solution: This is normal if content is identical

---

## Validation Errors

### Symptom: "Policy validation failed" errors

**Common validation errors:**

#### 1. Missing required fields

```
ERROR Policy must have at least one rule
```

**Solution:** Add at least one rule to the `rules:` array

#### 2. Invalid rule structure

```
ERROR Rule missing required field: conditions
```

**Solution:** Ensure each rule has `name`, `conditions`, and `actions`

#### 3. Invalid operator

```
ERROR Unknown operator: "equals"
```

**Solution:** Use valid operators: `==`, `!=`, `>`, `<`, `>=`, `<=`, `in`, `not_in`, `contains`, `matches_regex`

#### 4. Type mismatch

```
ERROR Value type mismatch for operator
```

**Solution:** Ensure value types match operator expectations
- `in`/`not_in`: value must be array
- `==`/`!=`: value must be scalar
- `matches_regex`: value must be string

**Enable strict validation:**

```yaml
policy:
  validation:
    enabled: true
    strict: true  # Fail on any validation error
```

---

## Performance Issues

### Symptom: Slow policy loading (> 100ms)

**Benchmarks (expected performance):**

- Single file load: < 50ms
- Directory with 10 files: < 100ms
- Policy Get operation: < 1µs (microsecond)
- Reload operation: < 50ms

**If seeing slower performance:**

```bash
# 1. Profile policy loading
go test -bench=BenchmarkPolicyManager_LoadPolicies -benchmem ./pkg/policy/manager/...

# 2. Check file sizes
find /path/to/policies -name "*.yaml" -exec ls -lh {} \;

# 3. Monitor system resources
top -p $(pgrep mercator)
```

**Common causes:**

- **Large files**: Files approaching 10MB limit
  - Solution: Split into multiple smaller files

- **Too many files**: Hundreds of policy files
  - Solution: Consolidate or use policy composition

- **Slow filesystem**: Network storage, encrypted filesystem
  - Solution: Use local SSD storage for policies

- **Complex includes**: Deep include chains (not yet implemented)
  - Solution: Flatten include structure

### Symptom: High memory usage

**Expected memory:**
- Baseline: < 100MB
- Under load: < 1GB

**If exceeding:**

```bash
# Check memory usage
ps aux | grep mercator

# Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Solutions:**
- Check for policy leaks (old versions not being GC'd)
- Reduce policy file count
- Review custom policy size

---

## File System Issues

### Symptom: "Failed to access policy path"

**Check:**

```bash
# 1. Path exists
ls -la /path/to/policies

# 2. Full path resolution
readlink -f /path/to/policies

# 3. Parent directory permissions
ls -la $(dirname /path/to/policies)
```

### Symptom: Symlink issues

**Default behavior:** Symlinks are followed

**To disable:**

```go
loaderConfig := &PolicyLoaderConfig{
    FollowSymlinks: false,
}
```

**Symlink loop detection:**

The loader detects and prevents symlink loops automatically.

```
ERROR Symlink loop detected at /path/to/policies/loop
```

### Symptom: "UTF-8 encoding validation failed"

**Cause:** Policy file contains non-UTF-8 characters

**Solution:**

```bash
# Check file encoding
file -i /path/to/policy.yaml

# Convert to UTF-8
iconv -f ISO-8859-1 -t UTF-8 policy.yaml > policy-utf8.yaml
```

---

## Configuration Problems

### Symptom: "config cannot be nil"

**Cause:** PolicyConfig not initialized

**Solution:**

```go
cfg := &config.PolicyConfig{
    Mode:     "file",
    FilePath: "/path/to/policy.yaml",
    Validation: config.PolicyValidationConfig{
        Enabled: true,
    },
}
```

### Symptom: "parser cannot be nil" or "validator cannot be nil"

**Cause:** Missing required dependencies

**Solution:**

```go
mgr, err := NewPolicyManager(
    cfg,
    parser.NewParser(),      // Must provide parser
    validator.NewValidator(), // Must provide validator
    logger,                   // Optional, can be nil
)
```

---

## Debugging Tips

### Enable Debug Logging

```go
import "log/slog"

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

mgr, err := NewPolicyManager(cfg, parser, validator, logger)
```

### Check Policy Version

```go
version := mgr.GetPolicyVersion()
fmt.Printf("Current policy version (hash): %s\n", version)
```

### Inspect Loaded Policies

```go
policies := mgr.GetAllPolicies()
for _, policy := range policies {
    fmt.Printf("Policy: %s (version %s), Rules: %d\n",
        policy.Name, policy.Version, len(policy.Rules))
}
```

### Test Individual Policy Files

```bash
# Lint a policy
mercator lint /path/to/policy.yaml

# Test policy against sample request
mercator test /path/to/policy.yaml --request sample.json

# Validate YAML syntax
python3 -c "import yaml; print(yaml.safe_load(open('policy.yaml')))"
```

### Monitor File Watcher Events

Set log level to DEBUG to see all file system events:

```
DEBUG File event detected path=/path/to/policy.yaml op=WRITE
DEBUG Debouncing file event
INFO Triggering policy reload
```

### Common Log Patterns

**Successful operation:**
```
INFO Loading policies mode=file path=/path/to/policy.yaml
INFO Policies loaded successfully count=1 version=abc123 duration_ms=15
```

**Failed operation:**
```
INFO Loading policies mode=file path=/path/to/policy.yaml
ERROR Failed to load policies error="..." duration_ms=5
```

**Hot-reload:**
```
INFO File watcher started path=/path/to/policies
INFO Triggering policy reload path=/path/to/policy.yaml
INFO Reloading policies
INFO Policies reloaded successfully count=1
```

### Check System Limits (Linux)

```bash
# Check inotify limits
cat /proc/sys/fs/inotify/max_user_watches
cat /proc/sys/fs/inotify/max_user_instances

# Increase if needed
sudo sysctl fs.inotify.max_user_watches=524288
```

---

## Getting Help

If issues persist:

1. **Collect diagnostic information:**
   ```bash
   # System info
   uname -a

   # File permissions
   ls -la /path/to/policies

   # Configuration
   cat config.yaml | grep -A 10 "policy:"

   # Recent logs (last 100 lines)
   tail -100 mercator.log
   ```

2. **Enable verbose logging and reproduce issue**

3. **Check GitHub issues:** https://github.com/mercator-hq/jupiter/issues

4. **File a bug report with:**
   - Mercator version
   - Operating system
   - Policy file samples (sanitized)
   - Full error logs
   - Steps to reproduce

---

## Quick Reference

### Configuration

```yaml
policy:
  mode: file                    # Only "file" mode supported currently
  file_path: ./policies/        # File or directory path
  watch: true                   # Enable hot-reload
  validation:
    enabled: true               # Validate policies on load
    strict: false               # Fail on any validation error
```

### File Requirements

- ✅ File size: < 10MB
- ✅ Encoding: UTF-8
- ✅ Extension: `.yaml` or `.yml`
- ✅ Permissions: Readable by process
- ✅ Valid YAML syntax
- ✅ Required fields: `mpl_version`, `name`, `version`, `rules`

### Performance Targets

- Policy load: < 100ms
- Policy reload: < 50ms
- Policy get: < 1µs
- Memory baseline: < 100MB

### Error Recovery

The policy manager implements error recovery - if a reload fails, the previous policies remain active. No downtime during failed reloads.
