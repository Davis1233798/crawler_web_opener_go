# Implementation Plan - Enforce Proxy Validation

The goal is to ensure that we only run the browser batch if the proxy is confirmed to be working. Currently, the IP check is optional/informational. We will make it strict: if the IP check fails, the batch aborts.

## User Review Required
> [!IMPORTANT]
> This change makes the proxy IP check mandatory. If `api.ipify.org` cannot be reached via the proxy, the batch will fail immediately. This prevents wasting resources on broken proxies.

## Proposed Changes

### Browser Package
#### [MODIFY] [browser.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/browser/browser.go)
- Update `RunBatch` method.
- Change the IP check logic:
    - If `p != nil` (proxy is used), attempt to fetch IP.
    - If fetch fails, return error immediately (do not launch browser).
    - If fetch succeeds, log the IP and proceed.

## Verification Plan

### Automated Tests
- None (Manual verification required as this depends on remote network conditions).

### Manual Verification
- Deploy to remote server.
- Run crawler.
- If proxy is bad, expect "Failed to check IP via proxy" and immediate batch failure (no browser launch).
- If proxy is good, expect "Connected via Proxy IP: X.X.X.X" and normal execution.
