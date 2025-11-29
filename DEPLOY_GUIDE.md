# 自動化爬蟲部署指南 (No-Proxy & Auto-Destruct)

本指南說明如何部署「無 Proxy 且自動銷毀」版本的爬蟲系統。此版本專為 GCP 環境設計，執行一次任務後會自動刪除 VM 以節省成本。

## 功能特點

1.  **無 Proxy 模式 (No-Proxy Mode)**:
    -   系統預設不使用任何 Proxy，直接使用 VM 的 IP 進行連線。
    -   略過所有 Proxy 列表的下載與檢查流程。

2.  **自動銷毀 (Auto-Destruct)**:
    -   **Run-Once**: 爬蟲執行完指定數量的任務後會自動停止。
    -   **Self-Destruct**: 任務完成後，程式會呼叫 GCP API (`gcloud compute instances delete`) 將自己刪除。

3.  **整合式鏡像 (Integrated Image)**:
    -   單一 Docker 鏡像包含 `crawler` (爬蟲程式) 與 `gcp_runner` (控制腳本)。
    -   內建 `gcloud` CLI 工具。

---

## 部署步驟

### 1. 構建並推送 Docker 鏡像

在您的開發環境中，構建 Docker 鏡像並推送到 Google Container Registry (GCR)。

```bash
# 請將 YOUR_PROJECT_ID 替換為您的 GCP 專案 ID
docker build -t gcr.io/YOUR_PROJECT_ID/crawler-integrated .
docker push gcr.io/YOUR_PROJECT_ID/crawler-integrated
```

### 2. 啟動主控端 VM (Master VM)

主控端 VM 負責發送指令來建立其他的 Worker VMs。

**權限要求**: 主控端 VM 的 Service Account 必須擁有 `Compute Admin` (或至少能建立/刪除實例) 的權限。

```bash
gcloud compute instances create-with-container crawler-master \
    --project YOUR_PROJECT_ID \
    --zone us-central1-a \
    --container-image gcr.io/YOUR_PROJECT_ID/crawler-integrated \
    --scopes https://www.googleapis.com/auth/cloud-platform
```

### 3. 執行自動化任務

連線到主控端 VM 並執行控制腳本。

```bash
# 1. SSH 進入主控端 VM
gcloud compute ssh crawler-master --project YOUR_PROJECT_ID --zone us-central1-a

# 2. 進入 Docker 容器
# 先找出容器 ID
docker ps
# 進入容器 (假設 ID 為 c12345)
docker exec -it c12345 /bin/bash

# 3. 執行部署指令
/app/gcp_runner \
  -image gcr.io/YOUR_PROJECT_ID/crawler-integrated \
  -project YOUR_PROJECT_ID \
  -zone us-central1-a \
  -count 10 \
  -run-once
```

**參數詳解**:
- `-image`: 您的 Docker 鏡像路徑 (必須與步驟 1 推送的路徑一致)。
- `-project`: 您的 GCP 專案 ID。
- `-zone`: 欲建立 Worker VM 的區域 (預設 us-central1-a)。
- `-count`: 要建立的 Worker VM 數量 (例如 10 台)。
- `-run-once`: **關鍵參數**。啟用此參數後，Worker VM 會在執行完任務後自動自我銷毀。

---

## 本地開發與測試

若要在本地測試 `gcp_runner` 的指令生成 (Dry Run)：

```bash
go run cmd/gcp_runner/main.go \
  -image test-image \
  -project test-project \
  -count 1 \
  -run-once \
  -dry-run
```

這會印出將要執行的 `gcloud` 指令，而不會真的建立 VM。
