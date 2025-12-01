# VLESS Connection Fix Walkthrough

I have implemented a fix to prevent the crawler from using the blocked domain-based VLESS link and to ensure SNI is preserved when using fetched IPs.

## Changes
- Modified `internal/proxy/proxy.go`:
    - Added `RemoveProxy` method to remove the base link.
    - Updated `UpdateProxiesFromIPs` to preserve `sni` and `host` parameters from the original link when replacing the address with an IP.
- Updated `internal/proxy/proxy_test.go`:
    - Added `TestRemoveProxy` to verify removal logic.
    - Added `TestUpdateProxiesFromIPs_SNI` to verify SNI preservation.

## Verification Results

### Remote Unit Test
I ran the unit tests on the remote server (`instance1`) to verify the logic works in the target environment.

**Command:**
```bash
/usr/local/go/bin/go test -v ./internal/proxy/...
```

**Output:**
```
=== RUN   TestRemoveProxy
--- PASS: TestRemoveProxy (0.00s)
=== RUN   TestUpdateProxiesFromIPs_SNI
2025/12/01 17:57:19 Started VLESS adapter at 127.0.0.1:36441
2025/12/01 17:57:19 Added 1 new proxies to pool. Total: 1
--- PASS: TestUpdateProxiesFromIPs_SNI (0.00s)
PASS
ok      github.com/Davis1233798/crawler-go/internal/proxy       0.010s
```

Both tests passed. This confirms:
1. The base link is correctly removed.
2. The new links generated from IPs correctly include `sni` and `host` parameters derived from the original domain.

## Next Steps
1. **Deploy & Run**: The changes are already pushed to the `feature/vless` branch and pulled to the remote server.
2. **Monitor**: Run the crawler on the remote server. The `connection reset` errors should be resolved.
   ```bash
   cd ~/crawler_web_opener_go
   go build -o crawler cmd/crawler/main.go
   ./crawler
   ```
