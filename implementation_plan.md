# Implementation Plan - Optimize Proxy Management

The goal is to prevent "excessive requests" and "mass IP failures" by limiting the number of IPs we process and verifying them in the background before use.

## User Review Required
> [!IMPORTANT]
> I will limit the number of fetched IPs to 50 per cycle to prevent rate limiting. I will also implement a background verification process so the crawler only picks up pre-verified proxies.

## Proposed Changes

### Proxy Package
#### [MODIFY] [ip_fetcher.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/ip_fetcher.go)
- Update `FetchPreferredIPs` to accept a `limit` argument (e.g., 50).
- Randomly select `limit` IPs from the fetched list if it exceeds the limit.

#### [MODIFY] [proxy.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/proxy.go)
- Add `unverifiedProxies` channel or list to `MemoryProxyPool`.
- Update `AddProxies` to add to `unverifiedProxies` instead of `workingProxies`.
- Implement `startBackgroundVerifier` in `Initialize`:
    - continuously pulls from `unverifiedProxies`.
    - checks IP (using `checkProxy`).
    - if good, adds to `workingProxies`.
    - if bad, discards.
    - respects a concurrency limit (e.g., 5 workers).

## Verification Plan

### Automated Tests
- `TestFetchPreferredIPs_Limit`: Verify it returns max N IPs.
- `TestBackgroundVerifier`: Verify that added proxies eventually appear in `workingProxies` if valid.

### Manual Verification
- Deploy to remote server.
- Monitor logs.
- Expect to see "Background verifier: Verified X proxies" logs.
- Expect `RunBatch` to proceed smoothly with valid proxies.
