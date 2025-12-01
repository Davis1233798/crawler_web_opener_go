# Implementation Plan - Remove Base VLESS Link

The goal is to prevent the crawler from using the original domain-based VLESS link, which causes `connection reset by peer` errors on the remote server. We will modify the proxy pool to remove this link once preferred IPs are successfully fetched.

## User Review Required
> [!IMPORTANT]
> This change causes the "base" VLESS link (defined in `.env`) to be removed from the active proxy pool once IPs are fetched. This is intended behavior to avoid using the blocked domain.

## Proposed Changes

### Proxy Package
#### [MODIFY] [proxy.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/proxy.go)
- Add `RemoveProxy(proxyStr string)` method to `MemoryProxyPool`.
- Update `UpdateProxiesFromIPs` to call `RemoveProxy(baseLink)` after adding new proxies.

## Verification Plan

### Automated Tests
- None (Manual verification required as this depends on remote network conditions).

### Manual Verification
- The user will need to deploy the changes to the remote server.
- Run the crawler and observe logs.
- Confirm that `[Error] ... failed to dial to ...workers.dev` stops appearing.
- Confirm that "Removed base VLESS adapter" log appears.
