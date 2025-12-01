# Implementation Plan - Fix IP Validation & Address Xray Warning

The goal is to resolve two issues identified in the logs:
1.  `failed to dial to Last:443`: Caused by parsing invalid text ("Last") as an IP.
2.  `WebSocket transport ... is deprecated`: Xray warning about WS transport.

## User Review Required
> [!IMPORTANT]
> I will add strict IP validation to ignore non-IP strings. I will also attempt to update the Xray config to silence or resolve the deprecation warning, though this depends on the server's supported protocols.

## Proposed Changes

### Proxy Package
#### [MODIFY] [ip_fetcher.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/ip_fetcher.go)
- Import `net`.
- Add `isValidIP(ip string) bool` helper.
- In `FetchPreferredIPs`, validate each extracted string. Discard if not a valid IP.

### VLESS Package
#### [MODIFY] [vless.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/vless.go)
- Investigate Xray config generation.
- The warning suggests migrating to `http` transport (XHTTP).
- However, since we don't control the server, we might be stuck with `ws`.
- I will check if we can explicitly set `path` or `headers` to satisfy Xray, or if we should just acknowledge it.
- **Action**: I will try to clean up the `wsSettings` to ensure it's compliant with recent Xray versions (e.g., ensure `Host` header is set correctly in `headers` if needed, though `host` field is preferred now).
- Actually, the warning says "migrated to XHTTP H2 & H3". This implies `ws` type itself is being deprecated in favor of `http` with `upgrade`.
- Since we can't change the server, we might just have to live with the warning, OR we can try to use `http` transport with `Upgrade: websocket` if Xray supports that mapping.
- **Decision**: For now, I will focus on the IP fix. I will make a minor adjustment to `vless.go` to ensure `host` is set correctly, but I might not be able to remove the warning if the server requires `ws`.

## Verification Plan

### Automated Tests
- `TestFetchPreferredIPs_Validation`: Verify "Last" is ignored.

### Manual Verification
- Deploy and run.
- Confirm "Last:443" error is gone.
- Check if EOFs persist (which would indicate the Xray warning/protocol is indeed the issue).
