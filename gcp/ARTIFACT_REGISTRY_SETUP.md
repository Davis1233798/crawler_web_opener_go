# 如何在 GCP Console 建立 Artifact Registry

請依照以下步驟在 Google Cloud Console 建立存放區，以配合部屬腳本使用。

## 步驟 1：進入 Artifact Registry
1.  開啟 [Google Cloud Console](https://console.cloud.google.com/)。
2.  在上方搜尋列輸入 **"Artifact Registry"** 並點選進入。
3.  點擊上方的 **「+ 建立存放區」 (+ CREATE REPOSITORY)** 按鈕。

## 步驟 2：填寫設定 (重要！)
請務必依照以下設定填寫，否則腳本會找不到存放區：

*   **名稱 (Name)**: `crawler-repo`
    *   *(請完全一致，不要改名)*
*   **格式 (Format)**: 選擇 **Docker**
*   **模式 (Mode)**: 選擇 **Standard** (標準)
*   **位置類型 (Location type)**: 選擇 **Region** (地區)
*   **地區 (Region)**: 選擇 **us-central1 (Iowa)**
    *   *(腳本預設推送到此地區，若選其他地區需修改腳本)*

## 步驟 3：建立
1.  確認設定無誤後，點擊底部的 **「建立」(CREATE)** 按鈕。
2.  等待幾秒鐘，直到列表出現 `crawler-repo`。

## 步驟 4：執行部屬
回到您的 PowerShell，執行部屬指令：

```powershell
.\deploy.ps1 smart-axis-479318-d5
```
