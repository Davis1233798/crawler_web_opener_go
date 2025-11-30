# Crawler Walkthrough

## Overview
This Go-based crawler opens multiple browser tabs concurrently to simulate human activity on target websites. It supports multiple modes of operation including standard proxy rotation, direct connection (no-proxy), and VLESS proxy support.

## Prerequisites
- Go 1.21+
- Playwright (`go run cmd/crawler/main.go` will install browsers if missing, or run `go run github.com/playwright-community/playwright-go/cmd/playwright install`)
- **For VLESS Mode**: `xray` binary (e.g., `xray.exe`) must be present in the `crawler-go` directory or system PATH.
- **For GCP Deployment**: Terraform installed and GCP credentials configured.

## Configuration
Configuration is managed via `.env` file:
- `THREADS`: Number of concurrent browser instances.
- `DURATION`: Duration (seconds) to stay on each page.
- `HEADLESS`: `true` for headless mode, `false` for visible browser.

## Running the Crawler

### Standard Mode (Proxy Rotation)
1.  Ensure you are on the `main` branch: `git checkout main`
2.  Populate `proxies.txt` with your proxies (format: `ip:port:user:pass`).
3.  Run: `go run cmd/crawler/main.go`

### No-Proxy Mode
1.  Switch to the `no-proxy` branch:
    ```bash
    git checkout no-proxy
    ```
2.  Run: `go run cmd/crawler/main.go`
    - The crawler will connect directly to target sites without using any proxy.

### VLESS Support Mode
1.  Switch to the `vless-support` branch:
    ```bash
    git checkout vless-support
    ```
2.  **Important**: Create a file named `vless.txt` in the `crawler-go` directory and paste your VLESS URI into it (e.g., `vless://uuid@host:port...`).
3.  Ensure `xray.exe` (or `xray` binary) is in the `crawler-go` directory.
4.  Run: `go run cmd/crawler/main.go`
    - The crawler will automatically parse the VLESS URI, generate a config, start Xray, and route traffic through it.

## GCP Deployment (Infrastructure as Code)
This project uses Terraform to automate the deployment of a "Mother VM" which continuously spawns short-lived crawler instances.

1.  Switch to the `iac-terraform` branch:
    ```bash
    git checkout iac-terraform
    ```
2.  Navigate to the Terraform directory:
    ```bash
    cd terraform
    ```
3.  Initialize Terraform:
    ```bash
    terraform init
    ```
4.  Apply the configuration:
    ```bash
    terraform apply -var="project_id=YOUR_PROJECT_ID"
    ```
    - This will create an Artifact Registry, a Service Account, and the Mother VM.
    - The Mother VM will automatically start the controller loop in the background.

5.  **Verification**:
    - Use the output SSH command to connect to the Mother VM and check logs:
      ```bash
      gcloud compute ssh crawler-mother-vm --zone=us-central1-a --command="tail -f /var/log/crawler_loop.log"
      ```

## Verification
- **Logs**: Check the console output.
    - Standard: "Using proxy..."
    - No-Proxy: "Running in NO-PROXY mode"
    - VLESS: "Xray started on 127.0.0.1:10808"
- **Browser**: In headed mode (`HEADLESS=false`), visually verify the browser opens and navigates.
