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

您可以選擇使用 **Google Container Registry (GCR)** 或 **Docker Hub**。

#### 選項 A: 使用 GCR (推薦用於 GCP)
```bash
# 請將 YOUR_PROJECT_ID 替換為您的 GCP 專案 ID
docker build -t gcr.io/YOUR_PROJECT_ID/crawler-integrated .
docker push gcr.io/YOUR_PROJECT_ID/crawler-integrated
```

#### 選項 B: 使用 Docker Hub
```bash
# 請將 YOUR_USERNAME 替換為您的 Docker Hub 帳號
docker build -t YOUR_USERNAME/crawler-integrated .
docker push YOUR_USERNAME/crawler-integrated
```

### 2. 啟動主控端 VM (Master VM)

主控端 VM 負責發送指令來建立其他的 Worker VMs。

**權限要求**: 主控端 VM 的 Service Account 必須擁有 `Compute Admin` (或至少能建立/刪除實例) 的權限。

#### 若使用 GCR:
```bash
gcloud compute instances create-with-container crawler-master \
    --project YOUR_PROJECT_ID \
    --zone us-central1-a \
    --container-image gcr.io/YOUR_PROJECT_ID/crawler-integrated \
    --scopes https://www.googleapis.com/auth/cloud-platform
```

#### 若使用 Docker Hub:
```bash
gcloud compute instances create-with-container crawler-master \
    --project YOUR_PROJECT_ID \
    --zone us-central1-a \
    --container-image YOUR_USERNAME/crawler-integrated \
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
# 若使用 GCR:
/app/gcp_runner \
  -image gcr.io/YOUR_PROJECT_ID/crawler-integrated \
  -project YOUR_PROJECT_ID \
  -zone us-central1-a \
  -count 10 \
  -run-once

# 若使用 Docker Hub:
/app/gcp_runner \
  -image YOUR_USERNAME/crawler-integrated \
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

## 無人值守與持續運行模式 (Unattended & Continuous Mode)

若您希望系統能自動、持續地生成 VM 進行爬取，直到您手動停止為止，請使用此模式。

### 1. 使用 `-continuous` 參數

在 `gcp_runner` 中加入 `-continuous` 與 `-interval` 參數：

- `-continuous`: 啟用持續模式。程式會無限循環，每隔一段時間生成一批新的 Worker VM。
- `-interval`: 每批次之間的間隔秒數 (例如 300 秒)。

### 2. 背景執行 (Daemon Mode)

為了讓主控端在您斷開 SSH 連線後仍能繼續運作，請使用 Docker 的 `-d` (Detached) 模式啟動主控容器。

```bash
# 1. 啟動主控容器於背景 (Detached Mode)
docker run -d --name crawler-master-daemon \
  gcr.io/YOUR_PROJECT_ID/crawler-integrated \
  /app/gcp_runner \
  -image gcr.io/YOUR_PROJECT_ID/crawler-integrated \
  -project YOUR_PROJECT_ID \
  -zone us-central1-a \
  -count 5 \
  -continuous \
  -interval 600

# 2. 查看日誌
docker logs -f crawler-master-daemon

# 3. 停止運行
docker stop crawler-master-daemon
```

**運作流程**:
1.  主控容器啟動，執行 `gcp_runner`。
2.  `gcp_runner` 建立 5 台 Worker VM。
3.  Worker VM 啟動，執行爬蟲任務，完成後**自我銷毀**。
4.  `gcp_runner` 等待 600 秒。
5.  `gcp_runner` 再次建立 5 台新的 Worker VM... (無限循環)
6.  直到您執行 `docker stop` 為止。

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
