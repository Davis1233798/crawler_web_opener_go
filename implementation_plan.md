# Implementation Plan - Main Branch Features

## Goal
Implement daily IP usage limits (1 run/day), automatic fallback to free proxies, and specific banner clicking logic on `tsplsimulator.dpdns.org`.

## User Review Required
> [!IMPORTANT]
> The crawler will create `ip_usage.json` to track daily usage.
> `proxies.txt` will be the primary source. Once an IP is used, it won't be used again until 24 hours (or next calendar day) have passed.

## Proposed Changes

### Proxy Management
#### [MODIFY] [internal/proxy/proxy.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/proxy/proxy.go)
- Add `UsageMap` (map[string]time.Time) to `MemoryProxyPool`.
- Implement persistence for `UsageMap` (`ip_usage.json`).
- Update `GetProxy`:
  - Check if proxy IP was used today.
  - If yes, skip it.
  - If all `proxies.txt` IPs are used, trigger `FetchFreeProxies`.
- Add `CheckProxyFast` utilizing `httpbin.org/ip` with 3s timeout.

### Browser Automation
#### [MODIFY] [internal/browser/browser.go](file:///c:/Users/solidityDeveloper/go_projects/crawler_web_opener_go/internal/browser/browser.go)
- Update `RunBatch`:
  - Navigate to `https://tsplsimulator.dpdns.org/`.
  - Wait for banner `img[src*='webtrafic.ru']`.
  - Scroll and click.
  - specific for `tsplsimulator`.

## Verification Plan

### Manual Verification
1. **IP Limit**:
   - Run crawler.
   - Stop it.
   - Run again. check logs: "Skipping proxy X (used today)".
2. **Fallback**:
   - Empty `proxies.txt` (or mark all used).
   - Run crawler.
   - Check logs: "Fetching free proxies...".
3. **Clicking**:
   - Run with `Headless: false`.
   - Watch browser open, go to site, click banner, close.
