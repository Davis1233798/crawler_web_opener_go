# Terraform 從零開始教學

這份教學會帶您從安裝 Terraform 開始，直到成功部屬我們的爬蟲基礎設施。

## 第一步：安裝 Terraform

由於您的電腦尚未安裝 Terraform，請選擇以下其中一種方式安裝：

### 方法 A：使用 Chocolatey (推薦，如果您有安裝的話)
在 PowerShell (以系統管理員身分執行) 中輸入：
```powershell
choco install terraform
```

### 方法 B：手動下載
1.  前往 [Terraform 下載頁面](https://developer.hashicorp.com/terraform/downloads)。
2.  選擇 **Windows** 版本下載 (AMD64)。
3.  解壓縮下載的 zip 檔，會看到一個 `terraform.exe`。
4.  **關鍵步驟**：將 `terraform.exe` 放到一個系統路徑 (PATH) 包含的資料夾中 (例如 `C:\Windows\System32`，或是您自己建立一個工具資料夾並加到 PATH)。
5.  重新開啟 PowerShell，輸入 `terraform -version` 確認是否安裝成功。

## 第二步：登入 Google Cloud

Terraform 需要權限才能幫您建立資源。請執行以下指令產生憑證：

```powershell
gcloud auth application-default login
```
(這會跳出瀏覽器視窗，請登入您的 Google 帳號並允許授權)

## 第三步：初始化 (Init)

進入我們的 terraform 目錄：
```powershell
cd terraform
```

初始化 Terraform (這會下載必要的 Google Cloud 插件)：
```powershell
terraform init
```
*成功時會看到綠色的 "Terraform has been successfully initialized!"*

## 第四步：預覽計畫 (Plan)

在真正建立資源之前，我們先看看 Terraform 打算做什麼。
請將 `您的專案ID` 替換成真的 ID (例如 `smart-axis-479318-d5`)：

```powershell
terraform plan -var="project_id=您的專案ID"
```
*這會列出一長串 "+" 號，表示將要新增的資源 (API, Repository, VM)。*

## 第五步：執行部屬 (Apply)

確認計畫沒問題後，我們就來真的了！

```powershell
terraform apply -var="project_id=您的專案ID"
```
*Terraform 會再次列出計畫，並問您 `Do you want to perform these actions?`*
請輸入 **`yes`** 並按 Enter。

## 第六步：大功告成

等待幾分鐘後，您會看到綠色的 `Apply complete!`。
此時 Terraform 已經幫您：
1.  開啟了必要的 API。
2.  建立了 Artifact Registry。
3.  **啟動了母體 VM，並且該 VM 已經自動開始執行爬蟲任務了！**

您可以透過輸出的 `ssh_command` 連線進去看看，或是直接去 GCP Console 看 VM 是否在跑。

## (選用) 清除資源 (Destroy)

如果您不想玩了，想把所有東西刪掉以免扣款：

```powershell
terraform destroy -var="project_id=您的專案ID"
```
(一樣輸入 `yes` 確認)
