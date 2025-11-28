# Concurrent Crawler Walkthrough

## Overview
We have updated the Go crawler to support **concurrent page opening** in a single browser instance. This allows multiple target sites to be opened simultaneously in different tabs, sharing the same browser fingerprint and proxy.

## Changes
- **Concurrent Batch Processing**: The crawler now reads ALL URLs from `target_site.txt` and opens them at once in a single browser window (multiple tabs).
- **Human Simulation**: Each tab simulates human activity independently, including:
    - Random scrolling
    - Mouse movements (Bezier-like curves)
    - **Random Clicking**: Clicks on links, buttons, or submit inputs.
- **Proxy Integration**: Added support for `http://user:pass@host:port` proxy format and added the provided Nimbleway proxy.

## How to Run

### 1. Configure Targets
Edit `target_site.txt` and add the URLs you want to open. Each line is a separate tab.
```text
https://example.com
https://google.com
https://bing.com
```

### 2. Configure Environment
Ensure `.env` has the desired settings:
```ini
HEADLESS=false  # Set to true for headless mode
DURATION=30     # Duration to keep pages open (seconds)
THREADS=1       # Number of concurrent BROWSER instances (usually 1 is enough if opening many tabs)
```

### 3. Run the Crawler
```bash
go run cmd/crawler/main.go
```

## Verification
- **Visual Check**: Set `HEADLESS=false`. You should see a Chrome window open with multiple tabs corresponding to your `target_site.txt` entries.
- **Activity**: Switch between tabs to observe mouse movements and scrolling.
- **Logs**: Check the terminal output for "Opening X tabs in parallel..." and "Batch completed successfully".
