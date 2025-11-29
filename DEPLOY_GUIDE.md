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
  -run-once \
  -continuous \
  -interval 300

# 若使用 Docker Hub:
/app/gcp_runner \
  -image YOUR_USERNAME/crawler-integrated \
  -project YOUR_PROJECT_ID \
  -zone us-central1-a \
  -count 10 \
  -run-once \
  -continuous \
  -interval 300
```

**參數詳解**:
- `-image`: 您的 Docker 鏡像路徑 (必須與步驟 1 推送的路徑一致)。
- `-project`: 您的 GCP 專案 ID。
- `-zone`: 欲建立 Worker VM 的區域 (預設 us-central1-a)。
- `-count`: 要建立的 Worker VM 數量 (例如 10 台)。
- `-run-once`: **關鍵參數**。啟用此參數後，Worker VM 會在執行完任務後自動自我銷毀。
- `-continuous`: **無人值守模式**。啟用後，程式會無限循環建立 VM。
- `-interval`: 在連續模式下，每批次之間的等待時間 (秒)。

## 關於 Docker Hub 與 OS

Docker 鏡像本身就包含了作業系統 (Base Image)。本專案使用 `mcr.microsoft.com/playwright:v1.40.0-jammy` 作為基底，它是基於 **Ubuntu 22.04 LTS (Jammy Jellyfish)** 的。

當您將鏡像推送到 Docker Hub 並在 GCP 上使用時，GCP 會下載這個包含完整 Ubuntu OS 環境的鏡像來啟動容器。因此，您不需要擔心底層 VM 的 OS 設定，一切都在鏡像中定義好了。


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
