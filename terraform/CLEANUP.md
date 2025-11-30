# 如何停止與清除資源

當您不需要爬蟲繼續執行時，有兩種方式可以停止：

## 方法一：徹底清除 (推薦)
這會刪除所有資源 (母體 VM、Artifact Registry、網路設定)，確保**完全不會再產生費用**。

請在 `terraform` 目錄下執行：

```powershell
terraform destroy -var="project_id=smart-axis-479318-d5"
```
(當系統詢問時，輸入 `yes` 確認)

---

## 方法二：只暫停爬蟲 (保留資源)
如果您只是想暫時停止爬蟲，但想保留母體 VM 和 Docker Image (方便下次快速啟動)：

1.  **連線到母體 VM**：
    ```powershell
    gcloud compute ssh crawler-mother-vm --zone=us-central1-a --project=smart-axis-479318-d5
    ```

2.  **砍掉控制迴圈**：
    ```bash
    pkill -f controller_loop.sh
    ```
    *(這會停止派送新任務，但正在跑的爬蟲會執行到結束)*

3.  **下次要恢復時**：
    ```bash
    nohup ./controller_loop.sh smart-axis-479318-d5 300 > /var/log/crawler_loop.log 2>&1 &
    ```
