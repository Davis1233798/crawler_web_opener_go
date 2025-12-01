# Implementation Plan - Fix SNI in VLESS IP Update

The goal is to ensure that when we replace the VLESS address with a fetched IP, we preserve the original domain for SNI (Server Name Indication) and Host header. This is critical for VLESS connections behind CDNs (like Cloudflare) where the IP alone is insufficient.

## User Review Required
> [!IMPORTANT]
> This change modifies how VLESS links are generated from fetched IPs. It explicitly adds `sni` and `host` parameters if they are missing, using the original domain from the base link.

## Proposed Changes

### Proxy Package
#### [MODIFY] [proxy.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/proxy.go)
- Update `UpdateProxiesFromIPs` method.
- Before replacing `u.Host` with the IP, extract the original hostname.
- Check if `sni` query parameter is present. If not, set it to the original hostname.
- Check if `host` query parameter is present. If not, set it to the original hostname.
- Reconstruct the URL query string.

## Verification Plan

### Automated Tests
- Create a unit test `TestUpdateProxiesFromIPs_SNI` in `proxy_test.go`.
- Verify that a base link without `sni` gets `sni` added when an IP is injected.
- Verify that a base link *with* `sni` preserves the existing `sni`.

### Manual Verification
- Deploy to remote server.
- Run crawler and check if connection errors resolve.
