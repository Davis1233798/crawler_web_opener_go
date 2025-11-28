# Implementation Plan - Concurrent Multi-Tab Crawler

## Goal
Develop a new version of the crawler that opens all target sites simultaneously in a single browser instance (same fingerprint), with each page remaining open for 30 seconds while simulating human clicking actions. Support both headless and headed modes.

## User Review Required
> [!IMPORTANT]
> This change modifies the core behavior of the crawler. Instead of processing one URL per browser instance, it will open **ALL** URLs found in `target_site.txt` in a single browser instance (multiple tabs).
> **Performance Warning**: If `target_site.txt` contains many URLs (e.g., >20), opening them all in one browser might consume significant RAM and CPU.

## Proposed Changes

### `internal/browser`

#### [MODIFY] [browser.go](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/internal/browser/browser.go)
- Add `RunBatch(urls []string, p *proxy.Proxy, duration int) error` method to `BrowserBot`.
- In `RunBatch`:
    - Create a single Browser Context (one fingerprint).
    - Iterate through all provided `urls`.
    - For each URL, create a new `Page` and navigate to it.
    - Launch a goroutine for each page to perform `simulateActivity` concurrently.
    - Wait for all pages to complete their 30-second duration.
- Refactor `simulateActivity` (currently inside `Run`) into a reusable helper method that accepts a `Page`.
- Ensure `humanMouseMove` and clicking logic is applied to each page.

### `cmd/crawler`

#### [MODIFY] [main.go](file:///c:/Users/solidityDeveloper/crawler_web_opener/crawler-go/cmd/crawler/main.go)
- Modify the worker loop to pass `cfg.Targets` (all targets) to `bot.RunBatch` instead of a single random target.
- Adjust logging to reflect batch processing.

## Verification Plan

### Manual Verification
1.  **Setup**:
    - Edit `target_site.txt` to contain 3-5 test URLs (e.g., `https://example.com`, `https://google.com`, `https://bing.com`).
    - Set `HEADLESS=false` in `.env` (or env var) to visually verify.
    - Set `DURATION=30`.
2.  **Execution**:
    - Run the crawler: `go run cmd/crawler/main.go`.
3.  **Observation**:
    - Confirm a single browser window opens.
    - Confirm multiple tabs open (one for each URL).
    - Confirm mouse movements/clicks happen on the tabs (might be hard to see all at once, but switching tabs should show activity).
    - Confirm browser closes after ~30 seconds.
