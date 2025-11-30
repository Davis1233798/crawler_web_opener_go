# Terraform Infrastructure Setup

此目錄包含使用 Terraform 自動化建立 GCP 基礎設施的設定檔。

## 建立的資源
1.  **啟用 API**: `artifactregistry.googleapis.com`, `compute.googleapis.com`
2.  **Artifact Registry**: 建立 `crawler-repo` 存放區。
3.  **Service Account**: 建立母體 VM 專用的服務帳號。
4.  **Compute Instance**: 建立「母體 VM (`crawler-mother-vm`)」，並自動安裝 Git 和 Docker。

## 前置需求
1.  [安裝 Terraform](https://developer.hashicorp.com/terraform/downloads)
2.  安裝 gcloud CLI 並登入 (`gcloud auth login`, `gcloud auth application-default login`)

## 使用方法

1.  **初始化** (只需執行一次):
    ```bash
    terraform init
    ```

2.  **檢視計畫**:
    ```bash
    terraform plan -var="project_id=您的專案ID"
    ```

3.  **執行部屬**:
    ```bash
    terraform apply -var="project_id=您的專案ID"
    ```
    (輸入 `yes` 確認)

4.  **連線到母體 VM**:
    部屬完成後，會顯示 SSH 連線指令，例如：
    ```bash
    gcloud compute ssh crawler-mother-vm --zone=us-central1-a
    ```

5.  **銷毀資源** (若不再需要):
    ```bash
    terraform destroy -var="project_id=您的專案ID"
    ```
