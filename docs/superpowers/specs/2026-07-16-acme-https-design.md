# ACME/HTTPS Implementation with Acmetool in Nginx

**Date:** 2026-07-16  
**Author:** Claude (Sonnet 4.5)  
**Status:** Approved  
**Domain:** chat.dure.one  
**Contact Email:** nikescar@gmail.com

## Overview

Implement automated HTTPS certificate management using acmetool and Let's Encrypt within a custom nginx Docker container. The solution provides zero-downtime certificate issuance and renewal with graceful fallback to HTTP if certificate operations fail.

## Goals

1. Enable HTTPS for chat.dure.one with valid Let's Encrypt certificates
2. Automate certificate renewal (zero manual intervention)
3. Maintain HTTP functionality during certificate operations (zero downtime)
4. Test safely using Let's Encrypt staging before production
5. Persist certificates across container restarts
6. Work within resource constraints (<1GB RAM, slow I/O)

## Non-Goals

- Multi-domain support (single domain: chat.dure.one)
- Certificate pinning or advanced security features
- Wildcard certificates
- Custom CA or self-signed certificates

## Architecture

### Container Structure

```
mycart-nginx-dev (custom nginx:alpine + acmetool)
├── nginx           - Web server and reverse proxy
├── acmetool        - ACME client for certificate management
├── supercronic     - Lightweight cron daemon for Alpine
├── entrypoint.sh   - Startup orchestration
└── renew-certs.sh  - Certificate renewal script
```

### Key Design Decisions

1. **Dual-mode nginx configuration**: HTTP on port 80 for ACME challenges + application traffic; HTTPS on port 443 after certificate issuance
2. **Graceful degradation**: If certificate issuance fails, nginx continues serving HTTP (no downtime)
3. **Staging-first workflow**: Test with Let's Encrypt staging environment, switch to production via environment variable
4. **Volume-mounted certificates**: Store in `./docker/nginx/certs` to persist across container recreation
5. **Zero-trust entrypoint**: Validate certificate state on every container start, handle missing certificates gracefully
6. **Lightweight dependencies**: acmetool (~3MB) + supercronic (~2MB) minimize resource usage

### File Structure

```
docker/nginx/
├── Dockerfile              # Custom nginx:alpine image with acmetool
├── nginx.conf              # Dual HTTP/HTTPS configuration
├── nginx-http-only.conf    # Fallback configuration (no SSL)
├── entrypoint.sh          # Startup orchestration script
├── renew-certs.sh         # Certificate renewal script
└── certs/                 # Certificate storage (volume mount)
    └── chat.dure.one/     # Per-domain certificate directory
```

## Components

### 1. Dockerfile

Custom nginx:alpine image extending the official image:

**Additions:**
- `acmetool` - ACME v2 client for Let's Encrypt
- `curl` - Health checks and testing
- `supercronic` - Container-friendly cron daemon

**Why supercronic?** Designed for containers, logs to stdout (Docker-friendly), more reliable than busybox crond in containerized environments.

**Image size impact:** +~5-10MB (nginx:alpine base ~25MB → ~35MB final)

### 2. entrypoint.sh

Orchestrates container startup with the following sequence:

```bash
1. Validate environment variables (DOMAIN, EMAIL, ACME_STAGING)
2. Initialize acmetool if first run:
   - Create ACME account with EMAIL
   - Configure staging or production endpoint based on ACME_STAGING
3. Check certificate status for DOMAIN:
   - Missing? → Issue new certificate using HTTP-01 challenge
   - Exists? → Validate expiry, renew if <30 days remaining
4. Copy certificates to nginx directory (/etc/nginx/certs/)
5. Select nginx configuration:
   - Certificates available? → Use full HTTPS config
   - Certificates missing? → Use HTTP-only fallback config
6. Start supercronic with renewal schedule (daily at 2 AM)
7. Start nginx in foreground (PID 1)
```

**Environment Variables:**
- `DOMAIN` (required) - Domain name for certificate (default: chat.dure.one)
- `EMAIL` (required) - Contact email for Let's Encrypt (default: nikescar@gmail.com)
- `ACME_STAGING` (optional) - Use staging environment (true/false, default: true)

**Exit behavior:** If nginx fails to start, container exits with error code (Docker restart policy handles recovery).

### 3. nginx.conf

Dual-mode configuration supporting HTTP-only and HTTPS modes:

**HTTP Mode (port 80):**
- Serve ACME challenges from `/var/www/acme-challenge/`
- Proxy all other traffic to `mycart-dev:8080`

**HTTPS Mode (port 443, after certificate issuance):**
- TLS 1.2+ only
- Modern cipher suite (Mozilla Intermediate profile)
- HSTS header (max-age=31536000)
- Proxy to `mycart-dev:8080`

**HTTP Redirect (after HTTPS enabled):**
- Port 80 redirects to HTTPS (301)
- Exception: `.well-known/acme-challenge/` always served over HTTP

**Configuration validation:** `nginx -t` runs before any reload to prevent broken configs.

### 4. renew-certs.sh

Called daily by cron, handles certificate renewal:

```bash
1. Run acmetool reconcile (checks all certificates)
2. If renewal occurred:
   - Copy new certificates to /etc/nginx/certs/
   - Test nginx configuration (nginx -t)
   - Reload nginx gracefully (nginx -s reload)
   - Log success
3. If renewal failed:
   - Log error with details
   - Check days until expiry
   - If <7 days, log URGENT warning
4. Exit 0 (don't fail cron job on errors)
```

**Cron schedule:** Daily at 2 AM container time (low-traffic period)

**Why daily checks?** Let's Encrypt best practice. Acmetool only renews when <30 days remain, so daily checks are lightweight (just a date comparison).

## Data Flow

### Initial Certificate Issuance

```
Container Start → entrypoint.sh
    ↓
Check: ACME account exists?
    ├─ No → acmetool quickstart --server [staging|prod] --email EMAIL
    └─ Yes → Continue
    ↓
Check: Certificate exists for DOMAIN?
    ├─ No → Issue certificate ──┐
    └─ Yes → Validate expiry     │
                                ↓
            acmetool want DOMAIN
                    ↓
            Start HTTP server on port 80
                    ↓
            Let's Encrypt sends HTTP-01 challenge
                    ↓
            Write challenge token to /var/www/acme-challenge/TOKEN
                    ↓
            LE validates http://DOMAIN/.well-known/acme-challenge/TOKEN
                    ↓
            Certificate issued → /var/lib/acme/live/DOMAIN/
                    ↓
            Copy to /etc/nginx/certs/DOMAIN/
                    ↓
Start nginx with HTTPS configuration
```

**Challenge method:** HTTP-01 (requires port 80 accessible)

**Certificate location:** 
- acmetool stores in `/var/lib/acme/live/DOMAIN/`
- Copied to `/etc/nginx/certs/DOMAIN/` for nginx consumption
- Both locations volume-mounted for persistence

### Certificate Renewal Flow

```
Cron triggers at 2 AM daily
    ↓
renew-certs.sh executes
    ↓
acmetool reconcile
    ↓
For each certificate:
    Check expiry date
    ├─ >30 days remaining → Skip
    └─ ≤30 days remaining → Renew
            ↓
    HTTP-01 challenge (same as initial)
            ↓
    New certificate issued → /var/lib/acme/live/
            ↓
    Copy to /etc/nginx/certs/
            ↓
    nginx -t (validate configuration)
            ↓
    nginx -s reload (graceful reload)
            ↓
    Log: "Certificate renewed for DOMAIN"
```

**Zero-downtime guarantee:** `nginx -s reload` performs graceful reload - existing connections complete normally while new connections use the new certificate.

**Monitoring:** All operations log to stdout/stderr, captured by `docker logs mycart-nginx-dev`

### Volume Mounts

```yaml
volumes:
  # ACME state and certificates (persistent)
  - ./docker/nginx/certs:/var/lib/acme:rw
  
  # Nginx configuration (read-only)
  - ./docker/nginx/nginx.conf:/etc/nginx/conf.d/default.conf:ro
  
  # ACME challenge directory (read-write, could be tmpfs)
  - ./docker/nginx/acme-challenges:/var/www/acme-challenge:rw
```

**Why separate ACME challenge volume?** Could use tmpfs for performance (challenges are ephemeral), while certificates require persistence.

## Error Handling

### 1. Certificate Issuance Failure

**Scenarios:**
- DNS not propagated to chat.dure.one
- Port 80 blocked by firewall
- Let's Encrypt rate limit exceeded
- Network connectivity issues

**Handling:**
```bash
if ! acmetool want "$DOMAIN"; then
    echo "⚠️ Certificate issuance failed for $DOMAIN"
    echo "Error: $(cat /var/lib/acme/log/latest.log | tail -5)"
    echo "Starting nginx in HTTP-only mode"
    cp /etc/nginx/nginx-http-only.conf /etc/nginx/conf.d/default.conf
fi
exec nginx -g 'daemon off;'
```

**Result:** Application stays online on HTTP. Admin reviews logs to diagnose issue. No service disruption.

**Recovery:** Fix underlying issue, restart container to retry.

### 2. Certificate Renewal Failure

**Scenarios:**
- Temporary LE outage
- Network issues during renewal window
- Challenge validation timeout

**Handling:**
```bash
if ! acmetool reconcile; then
    echo "⚠️ Certificate renewal failed at $(date)"
    echo "Error details: $(acmetool status 2>&1)"
    
    DAYS_LEFT=$(openssl x509 -in /var/lib/acme/live/$DOMAIN/cert -noout -enddate | \
                awk -F= '{print $2}' | xargs -I{} date -d {} +%s)
    NOW=$(date +%s)
    DAYS_UNTIL_EXPIRY=$(( ($DAYS_LEFT - $NOW) / 86400 ))
    
    echo "Days until expiry: $DAYS_UNTIL_EXPIRY"
    
    if [ $DAYS_UNTIL_EXPIRY -lt 7 ]; then
        echo "🚨 URGENT: Certificate expires in <7 days!"
        echo "Manual intervention required"
    fi
    
    exit 0  # Don't fail cron job
fi
```

**Result:** 
- Cron continues running daily attempts
- Logs show failure details and urgency
- Existing valid certificate continues working
- Alerts admin if <7 days remaining

**Recovery:** Most renewal failures are transient (network, LE API). Daily retries handle automatically. If persistent, admin investigates logs.

### 3. Nginx Reload Failure

**Scenarios:**
- Invalid certificate format
- Corrupted certificate file
- Configuration syntax error

**Handling:**
```bash
if ! nginx -t 2>/dev/null; then
    echo "⚠️ Nginx configuration test failed"
    echo "Configuration errors:"
    nginx -t 2>&1 | head -10
    echo "NOT reloading nginx - keeping current configuration"
    exit 1
fi

if ! nginx -s reload; then
    echo "⚠️ Nginx reload failed"
    echo "But nginx is still running with previous configuration"
    exit 1
fi

echo "✅ Nginx reloaded successfully with new certificate"
```

**Result:** Nginx continues running with old (but still valid) certificate. No downtime. Next renewal attempt may succeed.

**Recovery:** Script logs detailed errors. Admin can manually fix configuration or certificate if needed.

### 4. Resource Exhaustion

**Scenario:** OOM (out of memory) during certificate operations on resource-constrained server

**Mitigation:**
- Acmetool is lightweight (~5MB resident memory)
- Renewal runs at 2 AM (low traffic period)
- Challenge validation is I/O-bound, not memory-bound
- Cron job isolation prevents cascading failures

**Monitoring:**
```bash
# Check resource usage during operations
docker stats mycart-nginx-dev --no-stream
```

**Expected:** <50MB memory during normal operation, <70MB during renewal

### 5. Staging vs Production Mix-up

**Protection:**
```bash
if [ "$ACME_STAGING" = "true" ]; then
    echo "⚠️⚠️⚠️ Using Let's Encrypt STAGING environment ⚠️⚠️⚠️"
    echo "Certificates are NOT trusted by browsers (for testing only)"
    echo "Set ACME_STAGING=false for production certificates"
    ACME_URL="https://acme-staging-v02.api.letsencrypt.org/directory"
else
    echo "✅ Using Let's Encrypt PRODUCTION environment"
    echo "Issuing browser-trusted certificates"
    ACME_URL="https://acme-v02.api.letsencrypt.org/directory"
fi

acmetool quickstart --batch --agree-tos --server "$ACME_URL" --email "$EMAIL"
```

**Clear visual distinction** in logs prevents accidental production issuance during testing.

**Rate limit protection:** Staging environment has separate, much higher rate limits (for testing).

## Testing Strategy

### Pre-Implementation Tests (TDD Approach)

Create test suite before building implementation:

#### Test 1: HTTP Access (Baseline)
```bash
#!/bin/bash
# tests/test-http.sh

echo "Testing HTTP access to chat.dure.one..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://chat.dure.one/)

if [ "$RESPONSE" = "200" ]; then
    echo "✅ PASS: HTTP returns 200 OK"
    exit 0
else
    echo "❌ FAIL: HTTP returned $RESPONSE (expected 200)"
    exit 1
fi
```

**Expected:** Pass before and after HTTPS implementation (HTTP must always work)

#### Test 2: ACME Challenge Path
```bash
#!/bin/bash
# tests/test-acme-challenge.sh

echo "Testing ACME challenge path..."

# Create test token
TEST_TOKEN="test-$(date +%s)"
docker exec mycart-nginx-dev sh -c "echo 'test-content' > /var/www/acme-challenge/$TEST_TOKEN"

# Verify accessible via HTTP
CONTENT=$(curl -s http://chat.dure.one/.well-known/acme-challenge/$TEST_TOKEN)

if [ "$CONTENT" = "test-content" ]; then
    echo "✅ PASS: ACME challenge path accessible"
    exit 0
else
    echo "❌ FAIL: Got '$CONTENT' (expected 'test-content')"
    exit 1
fi
```

**Expected:** Pass (nginx serves challenge directory correctly)

#### Test 3: HTTPS Access (After Cert Issuance)
```bash
#!/bin/bash
# tests/test-https.sh

echo "Testing HTTPS access to chat.dure.one..."

# Test 1: HTTPS returns 200
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" https://chat.dure.one/)
if [ "$RESPONSE" != "200" ]; then
    echo "❌ FAIL: HTTPS returned $RESPONSE (expected 200)"
    exit 1
fi

# Test 2: HSTS header present
HSTS=$(curl -s -I https://chat.dure.one/ | grep -i "Strict-Transport-Security")
if [ -z "$HSTS" ]; then
    echo "❌ FAIL: HSTS header missing"
    exit 1
fi

# Test 3: TLS version check (1.2+)
TLS_VERSION=$(echo | openssl s_client -connect chat.dure.one:443 2>/dev/null | grep "Protocol" | awk '{print $3}')
if [[ "$TLS_VERSION" != "TLSv1.2" && "$TLS_VERSION" != "TLSv1.3" ]]; then
    echo "❌ FAIL: Weak TLS version $TLS_VERSION"
    exit 1
fi

echo "✅ PASS: HTTPS configured correctly"
exit 0
```

**Expected:** Pass after certificate issuance

#### Test 4: HTTP Redirect (After HTTPS Enabled)
```bash
#!/bin/bash
# tests/test-redirect.sh

echo "Testing HTTP → HTTPS redirect..."

# Test 1: HTTP redirects (301/302)
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://chat.dure.one/)
if [[ "$RESPONSE" != "301" && "$RESPONSE" != "302" ]]; then
    echo "❌ FAIL: Expected redirect, got $RESPONSE"
    exit 1
fi

# Test 2: Location header points to HTTPS
LOCATION=$(curl -s -I http://chat.dure.one/ | grep -i "Location:" | awk '{print $2}' | tr -d '\r\n')
if [[ "$LOCATION" != https://chat.dure.one/* ]]; then
    echo "❌ FAIL: Redirect location '$LOCATION' doesn't start with https://chat.dure.one/"
    exit 1
fi

# Test 3: ACME challenges NOT redirected
CHALLENGE_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://chat.dure.one/.well-known/acme-challenge/test)
if [[ "$CHALLENGE_RESPONSE" != "200" && "$CHALLENGE_RESPONSE" != "404" ]]; then
    echo "❌ FAIL: ACME challenge got redirect (should be 200/404)"
    exit 1
fi

echo "✅ PASS: HTTP redirects correctly, ACME challenges exempt"
exit 0
```

**Expected:** Pass after HTTPS enabled (HTTP redirects, except ACME paths)

#### Test 5: Certificate Validity
```bash
#!/bin/bash
# tests/test-cert-validity.sh

echo "Testing certificate validity..."

# Get certificate details
CERT_INFO=$(echo | openssl s_client -connect chat.dure.one:443 2>/dev/null | openssl x509 -noout -text)

# Check issuer (Let's Encrypt)
ISSUER=$(echo "$CERT_INFO" | grep "Issuer:" | grep -i "Let's Encrypt")
if [ -z "$ISSUER" ]; then
    echo "❌ FAIL: Certificate not issued by Let's Encrypt"
    exit 1
fi

# Check subject (chat.dure.one)
SUBJECT=$(echo "$CERT_INFO" | grep "Subject:" | grep "chat.dure.one")
if [ -z "$SUBJECT" ]; then
    echo "❌ FAIL: Certificate subject doesn't match chat.dure.one"
    exit 1
fi

# Check validity period
NOT_AFTER=$(echo | openssl s_client -connect chat.dure.one:443 2>/dev/null | openssl x509 -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$NOT_AFTER" +%s)
NOW_EPOCH=$(date +%s)
DAYS_REMAINING=$(( ($EXPIRY_EPOCH - $NOW_EPOCH) / 86400 ))

if [ $DAYS_REMAINING -lt 7 ]; then
    echo "⚠️ WARNING: Certificate expires in $DAYS_REMAINING days"
fi

if [ $DAYS_REMAINING -lt 0 ]; then
    echo "❌ FAIL: Certificate expired ${DAYS_REMAINING#-} days ago"
    exit 1
fi

echo "✅ PASS: Certificate valid (expires in $DAYS_REMAINING days)"
exit 0
```

**Expected:** Pass with valid date range (60-90 days typically)

### Staging Environment Validation

#### Phase 1: Staging Deployment
```bash
# 1. Deploy with staging environment
docker-compose -f docker-compose.simple.yml up -d

# 2. Wait for certificate issuance (check logs)
docker logs -f mycart-nginx-dev

# Expected log: "✅ Certificate issued for chat.dure.one (STAGING)"

# 3. Run all tests
cd tests
./test-http.sh
./test-acme-challenge.sh
./test-https.sh          # Will show browser warning (staging cert untrusted)
./test-redirect.sh
./test-cert-validity.sh

# 4. Verify staging certificate
echo | openssl s_client -connect chat.dure.one:443 2>/dev/null | grep "Issuer:"
# Expected: "Issuer: C = US, O = (STAGING) Let's Encrypt..."

# 5. Test renewal script manually
docker exec mycart-nginx-dev /renew-certs.sh
docker logs mycart-nginx-dev | tail -20
```

**Success criteria:** All tests pass, staging certificate issued, renewal script executes without errors.

#### Phase 2: Production Deployment
```bash
# 1. Update docker-compose environment
# Change: ACME_STAGING=false

# 2. Restart container
docker-compose -f docker-compose.simple.yml restart mycart-nginx-dev

# 3. Wait for production certificate issuance
docker logs -f mycart-nginx-dev
# Expected: "✅ Certificate issued for chat.dure.one (PRODUCTION)"

# 4. Re-run all tests
cd tests
./test-https.sh  # Should now show trusted certificate
./test-cert-validity.sh

# 5. Browser validation
# Open https://chat.dure.one in browser
# Verify green padlock, no warnings
# Check certificate details: issued by "Let's Encrypt"
```

**Success criteria:** Browser shows trusted certificate, no warnings.

### Renewal Testing

#### Manual Renewal Trigger
```bash
# Force renewal regardless of expiry date
docker exec mycart-nginx-dev sh -c "acmetool revoke chat.dure.one && acmetool want chat.dure.one"

# Watch renewal process
docker logs -f mycart-nginx-dev

# Expected sequence:
# 1. "Certificate revoked for chat.dure.one"
# 2. "Issuing new certificate..."
# 3. "✅ Certificate issued"
# 4. "Reloading nginx"
# 5. "✅ Nginx reloaded successfully"

# Verify new certificate
./tests/test-cert-validity.sh
```

#### Cron-based Renewal
```bash
# Verify cron job exists
docker exec mycart-nginx-dev cat /etc/cron.d/renew-certs
# Expected: "0 2 * * * /renew-certs.sh"

# Trigger cron manually (don't wait for 2 AM)
docker exec mycart-nginx-dev /renew-certs.sh

# Check logs
docker logs mycart-nginx-dev | grep -A 5 "Certificate renewal"

# Expected (if >30 days remaining):
# "Certificate for chat.dure.one has 60 days remaining (not renewing)"

# Or (if <30 days remaining):
# "Renewing certificate for chat.dure.one (15 days remaining)"
```

### Failure Mode Testing

#### Test 1: Port 80 Blocked
```bash
# Simulate blocked port
iptables -A INPUT -p tcp --dport 80 -j DROP

# Attempt certificate issuance
docker restart mycart-nginx-dev

# Expected behavior:
# - Logs show "ACME challenge failed: connection timeout"
# - Container starts in HTTP-only fallback mode
# - Application still accessible on HTTP
# - No crash, no downtime

# Cleanup
iptables -D INPUT -p tcp --dport 80 -j DROP
```

#### Test 2: Rate Limit Hit
```bash
# Simulate by using production endpoint without staging first
# (Don't actually do this - just document expected behavior)

# Expected behavior:
# - acmetool returns "Rate limit exceeded" error
# - Logs show clear error message with retry-after date
# - Container starts in HTTP-only fallback mode
# - Next renewal attempt respects rate limit window

# Recovery: Wait for rate limit window (1 hour for retries, 1 week for new certs)
```

#### Test 3: DNS Not Propagated
```bash
# Simulate by temporarily pointing DNS elsewhere
# (Test in staging only)

# Expected behavior:
# - ACME validation fails: "Domain validation failed"
# - Clear error: "chat.dure.one does not resolve to this server"
# - Container starts in HTTP-only fallback mode

# Recovery: Fix DNS, restart container
```

#### Test 4: Certificate Expiry Emergency
```bash
# Simulate by manually setting cert expiry to <7 days
# (Test script, not actual production)

# Trigger renewal
docker exec mycart-nginx-dev /renew-certs.sh

# Expected logs:
# "⚠️ Certificate for chat.dure.one expires in 5 days"
# "🚨 URGENT: Manual intervention may be required"
# "Attempting renewal..."
# [renewal process]
```

### Integration Test (Full End-to-End)

```bash
#!/bin/bash
# tests/integration-test.sh

set -e  # Exit on any error

echo "=== Integration Test: ACME/HTTPS Full Workflow ==="

echo "\n1. Starting with staging environment..."
export ACME_STAGING=true
docker-compose -f docker-compose.simple.yml up -d

echo "\n2. Waiting for staging certificate issuance (max 2 minutes)..."
timeout 120 sh -c 'until docker logs mycart-nginx-dev 2>&1 | grep -q "Certificate issued"; do sleep 5; done'

echo "\n3. Running test suite..."
./tests/test-http.sh
./tests/test-acme-challenge.sh
./tests/test-https.sh
./tests/test-redirect.sh
./tests/test-cert-validity.sh

echo "\n4. Testing manual renewal..."
docker exec mycart-nginx-dev /renew-certs.sh

echo "\n5. Switching to production environment..."
export ACME_STAGING=false
docker-compose -f docker-compose.simple.yml up -d --force-recreate

echo "\n6. Waiting for production certificate issuance (max 2 minutes)..."
timeout 120 sh -c 'until docker logs mycart-nginx-dev 2>&1 | grep -q "Certificate issued.*PRODUCTION"; do sleep 5; done'

echo "\n7. Running final test suite..."
./tests/test-https.sh
./tests/test-cert-validity.sh

echo "\n8. Browser validation..."
echo "Please open https://chat.dure.one in a browser and verify:"
echo "  - Green padlock icon"
echo "  - No certificate warnings"
echo "  - Certificate issued by 'Let's Encrypt'"
read -p "Press Enter to continue after browser verification..."

echo "\n✅ Integration test completed successfully!"
```

**Success criteria:** All automated tests pass, browser shows trusted certificate.

### Resource Monitoring

Monitor resource usage during certificate operations on constrained server:

```bash
# Monitor during certificate issuance
docker stats mycart-nginx-dev --no-stream

# Expected metrics:
# - Memory: <50MB normal, <70MB during issuance/renewal
# - CPU: <5% normal, <20% during operations
# - I/O: Minimal (ACME is network-bound)

# Long-term monitoring
docker stats mycart-nginx-dev --no-stream --format "table {{.Container}}\t{{.MemUsage}}\t{{.CPUPerc}}"

# Alert if memory >100MB (indicates problem)
```

### Test Coverage Summary

| Component | Tests | Pass Criteria |
|-----------|-------|---------------|
| HTTP Access | test-http.sh | Always returns 200 OK |
| ACME Challenges | test-acme-challenge.sh | .well-known path accessible |
| HTTPS Setup | test-https.sh | Valid TLS, HSTS header |
| HTTP Redirect | test-redirect.sh | Redirects except ACME paths |
| Certificate | test-cert-validity.sh | Valid LE cert, >7 days remain |
| Renewal | Manual + cron tests | Successful renewal, nginx reload |
| Failure Modes | 4 scenarios | Graceful degradation to HTTP |
| Integration | Full workflow | End-to-end staging → prod |
| Resources | Monitoring | <70MB memory during ops |

## Implementation Checklist

- [ ] Create `docker/nginx/Dockerfile` with acmetool + supercronic
- [ ] Write `docker/nginx/entrypoint.sh` with startup orchestration
- [ ] Write `docker/nginx/renew-certs.sh` with renewal logic
- [ ] Create `docker/nginx/nginx.conf` with dual HTTP/HTTPS config
- [ ] Create `docker/nginx/nginx-http-only.conf` as fallback
- [ ] Update `docker-compose.simple.yml` with custom image and volumes
- [ ] Create `tests/` directory with all test scripts
- [ ] Test in staging environment (ACME_STAGING=true)
- [ ] Validate all tests pass
- [ ] Switch to production (ACME_STAGING=false)
- [ ] Verify browser shows trusted certificate
- [ ] Monitor resource usage
- [ ] Document deployment process in README

## Deployment Process

### Initial Deployment (Staging)

```bash
# 1. Build custom nginx image
cd /srv/mycart
docker-compose -f docker-compose.simple.yml build nginx-dev

# 2. Start with staging environment
export ACME_STAGING=true
docker-compose -f docker-compose.simple.yml up -d

# 3. Monitor certificate issuance
docker logs -f mycart-nginx-dev

# 4. Run tests once "Certificate issued" appears
cd tests
./test-http.sh && ./test-https.sh && ./test-redirect.sh

# 5. Verify staging certificate in browser (will show warning - expected)
```

### Production Deployment

```bash
# 1. Update environment variable
# Edit docker-compose.simple.yml: ACME_STAGING=false

# 2. Recreate container with new environment
docker-compose -f docker-compose.simple.yml up -d --force-recreate nginx-dev

# 3. Monitor production certificate issuance
docker logs -f mycart-nginx-dev

# 4. Verify in browser (should show trusted cert, green padlock)
open https://chat.dure.one
```

### Maintenance

```bash
# Check renewal logs (run weekly)
docker logs mycart-nginx-dev | grep "renewal"

# Verify certificate expiry
./tests/test-cert-validity.sh

# Manual renewal trigger (if needed)
docker exec mycart-nginx-dev /renew-certs.sh

# Check cron job is running
docker exec mycart-nginx-dev ps aux | grep crond
```

## Security Considerations

1. **Private key protection**: Keys stored in `/var/lib/acme/keys/` (volume-mounted, not in image)
2. **TLS configuration**: Mozilla Intermediate profile (balance of security and compatibility)
3. **HSTS header**: Enabled with 1-year max-age after HTTPS proven stable
4. **Rate limiting**: Let's Encrypt staging protects against accidental quota exhaustion
5. **Email notifications**: Expiry warnings sent to nikescar@gmail.com
6. **Minimal attack surface**: Only acmetool, nginx, and supercronic in container

## Performance Considerations

1. **Resource usage**: Acmetool + supercronic add ~10MB memory overhead
2. **Certificate operations**: Renewal runs at 2 AM (low traffic)
3. **Nginx reload**: Graceful reload (<1s disruption to new connections, zero to existing)
4. **Challenge validation**: ~1-5 seconds per challenge (HTTP-01)
5. **I/O impact**: Minimal on resource-constrained system (ACME is network-bound)

## Rollback Plan

If HTTPS implementation causes issues:

```bash
# 1. Immediate rollback: revert to HTTP-only
docker exec mycart-nginx-dev cp /etc/nginx/nginx-http-only.conf /etc/nginx/conf.d/default.conf
docker exec mycart-nginx-dev nginx -s reload

# 2. Or: revert to old nginx image
docker-compose -f docker-compose.simple.yml stop nginx-dev
# Edit docker-compose.simple.yml: image: nginx:alpine
docker-compose -f docker-compose.simple.yml up -d nginx-dev

# 3. Investigate logs
docker logs mycart-nginx-dev > /tmp/nginx-debug.log
```

Application continues on HTTP with zero downtime.

## Future Enhancements (Out of Scope)

- Multi-domain support (wildcards or multiple subdomains)
- OCSP stapling for improved TLS performance
- Certificate pinning for advanced security
- Automatic testing of staging certificates before production deployment
- Webhook notifications for renewal failures
- Support for DNS-01 challenge (for wildcard certs)

## Questions and Answers

**Q: Why acmetool instead of certbot?**  
A: Acmetool is more lightweight (~3MB vs ~50MB), has better automation (reconcile command), and simpler configuration. Ideal for constrained environments.

**Q: What if Let's Encrypt is down during renewal?**  
A: Certificates valid for 90 days, renewal starts at 60 days. Multiple daily retry attempts provide ~30 days buffer even with extended LE outages.

**Q: Can I use this for multiple domains?**  
A: Current design is single-domain. Multi-domain requires minor modifications to entrypoint.sh (loop over DOMAINS env var).

**Q: What happens if container is destroyed and recreated?**  
A: Certificates persist in `./docker/nginx/certs` volume mount. New container finds existing valid certificates and uses them. No re-issuance needed.

**Q: How do I monitor renewal status?**  
A: `docker logs mycart-nginx-dev | grep renewal` shows all renewal attempts. Set up log monitoring/alerting if desired.

## References

- [acmetool documentation](https://github.com/hlandau/acmetool)
- [Let's Encrypt rate limits](https://letsencrypt.org/docs/rate-limits/)
- [Let's Encrypt staging environment](https://letsencrypt.org/docs/staging-environment/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [Docker best practices for containers](https://docs.docker.com/develop/dev-best-practices/)

---

**End of Design Document**
