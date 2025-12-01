# Debugging Remote VLESS Connection

- [x] Analyze the error logs and identify the root cause of `connection reset by peer` <!-- id: 0 -->
- [x] Review `internal/proxy/vless.go` and `internal/proxy/proxy.go` for connection logic <!-- id: 1 -->
- [x] Propose potential fixes (remove base link) <!-- id: 3 -->
- [x] Implement `RemoveProxy` and update `UpdateProxiesFromIPs` <!-- id: 5 -->
- [x] Deploy changes to remote server <!-- id: 6 -->
- [x] Verify the fix on remote server <!-- id: 4 -->
- [x] Implement SNI preservation in `UpdateProxiesFromIPs` <!-- id: 7 -->
- [x] Verify SNI fix with unit tests <!-- id: 8 -->
- [x] Deploy and verify on remote server <!-- id: 9 -->
- [ ] Implement strict proxy validation in `RunBatch` <!-- id: 10 -->
- [ ] Deploy and verify strict validation <!-- id: 11 -->
