# GCP 爬蟲部屬指南 (無 Proxy 版本)

本目錄包含將 `no-proxy` 版本爬蟲部屬到 Google Cloud Platform (GCP) 多個地區的腳本。

## 功能特點
- **多地區部屬**：自動在多個地區 (可於腳本中設定) 建立 VM 實例，以利用不同的外部 IP。
- **自動銷毀**：每個實例在執行完爬蟲任務後，會等待 60 秒並**自動刪除自己**，避免產生額外費用。
- **Docker 化**：使用包含 Playwright 依賴的 Docker 容器執行。

## 前置需求
1.  **Google Cloud Project**：您需要有一個 GCP 專案 ID。
2.  **gcloud CLI**：已安裝並完成驗證 (`gcloud auth login`)。
3.  **Docker**：本地已安裝 Docker 以便建置映像檔。
4.  **已啟用 API**：
    - Compute Engine API
    - Container Registry API (或 Artifact Registry)

## 使用方法

### Windows 使用者 (PowerShell)

1.  **進入目錄**：
    ```powershell
    cd gcp
    ```

2.  **執行部屬腳本**：
    請將 `YOUR_PROJECT_ID` 替換為您的 GCP 專案 ID。
    ```powershell
    .\deploy.ps1 YOUR_PROJECT_ID
    ```

### Linux / Mac 使用者 (Bash)

1.  **進入目錄**：
    ```bash
    cd gcp
    ```

2.  **執行部屬腳本**：
    ```bash
    chmod +x deploy.sh
    ./deploy.sh YOUR_PROJECT_ID
    ```

## 腳本執行流程
1.  **建置映像檔**：在本地建置 Docker Image。
2.  **推送映像檔**：將 Image 推送到 Google Container Registry (`gcr.io/YOUR_PROJECT_ID/crawler-no-proxy:latest`)。
3.  **建立 VM**：在設定的地區 (預設：`us-central1`, `europe-west1`, `asia-northeast1`) 建立 VM。
4.  **執行任務**：VM 啟動後會自動拉取 Image 並執行爬蟲。
5.  **自動銷毀**：任務完成後，VM 會自動執行刪除指令。

## 設定
- **地區 (Regions)**：編輯 `deploy.ps1` 或 `deploy.sh` 中的 `REGIONS` 變數來新增或移除地區。
- **爬蟲參數**：環境變數 (`THREADS`, `DURATION`) 已在腳本的 `metadata` (startup-script) 中設定。
