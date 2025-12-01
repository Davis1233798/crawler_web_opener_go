# Debugging Remote VLESS Connection

- [x] Analyze the error logs and identify the root cause of `connection reset by peer` <!-- id: 0 -->
- [x] Review `internal/proxy/vless.go` and `internal/proxy/proxy.go` for connection logic <!-- id: 1 -->
- [x] Propose potential fixes (remove base link) <!-- id: 3 -->
- [x] Implement `RemoveProxy` and update `UpdateProxiesFromIPs` <!-- id: 5 -->
- [x] Deploy changes to remote server <!-- id: 6 -->
- [x] Verify the fix on remote server <!-- id: 4 -->
- [ ] Implement SNI preservation in `UpdateProxiesFromIPs` <!-- id: 7 -->
- [ ] Verify SNI fix with unit tests <!-- id: 8 -->
- [ ] Deploy and verify on remote server <!-- id: 9 -->
