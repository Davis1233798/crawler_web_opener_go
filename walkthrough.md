# Ephemeral Xray Adapters Walkthrough

## Problem
The user requested a mechanism to "monitor Xray status, timely recycle on error, and recycle after correct execution".
The previous "Long-Lived Adapter" approach had risks of:
1.  **Zombie Processes**: If the app crashed or failed to cleanup, Xray instances might persist.
2.  **Stale State**: Long-running adapters might get into bad states (e.g. connection loops).
3.  **Resource Leaks**: Accumulation of unused adapters.

## Solution: Ephemeral Adapters
I refactored `MemoryProxyPool` to treat Xray adapters as **ephemeral resources**, tied strictly to the lifecycle of a single batch execution.

### Lifecycle
1.  **Initialize**:
    -   Loads VLESS config strings from disk.
    -   **Does NOT** start any Xray processes.
2.  **GetProxy (Start)**:
    -   Selects a free VLESS config.
    -   **Starts** a new Xray adapter instance on a random port.
    -   **Throttling**: If startup fails, sleeps 200ms and retries (max 5 times) to prevent CPU/Process storms.
    -   Returns the local SOCKS5 address.
    -   Maps `SocksAddr -> Adapter`.
3.  **RunBatch (Use)**:
    -   Crawler uses the local SOCKS5 proxy.
4.  **ReleaseProxy / MarkFailed (Recycle)**:
    -   **Closes** the Xray adapter immediately (kills the process).
    -   Removes it from active maps.
    -   Unmarks the VLESS config as busy.

### Code Changes
-   **`internal/proxy/proxy.go`**:
    -   Removed `vlessAdapters` (long-lived map).
    -   Added `activeAdapters` (short-lived map).
    -   Updated `GetProxy` to start adapters with **throttling**.
    -   Updated `ReleaseProxy` and `MarkFailed` to close adapters.
    -   Cleaned up `Initialize`, `AddProxies`, `replenish` to remove old startup logic.

## Verification
-   **Recycle on Error**: `MarkFailed` calls `adapter.Close()`.
-   **Recycle on Success**: `ReleaseProxy` calls `adapter.Close()`.
-   **Monitoring**: Process is only alive while being used. If `RunBatch` fails (e.g. Xray crash), `MarkFailed` ensures cleanup.
-   **Storm Prevention**: `GetProxy` backs off if adapters fail to start.

This architecture ensures that **1 Batch = 1 Xray Process**, guaranteeing a clean slate for every execution and preventing "dead process" accumulation.
