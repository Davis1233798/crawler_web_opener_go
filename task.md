# Debugging Remote VLESS Connection

- [x] Analyze the error logs and identify the root cause of `connection reset by peer` <!-- id: 0 -->
- [x] Review `internal/proxy/vless.go` and `internal/proxy/proxy.go` for connection logic <!-- id: 1 -->
- [x] Propose potential fixes (remove base link) <!-- id: 3 -->
- [ ] Implement `RemoveProxy` and update `UpdateProxiesFromIPs` <!-- id: 5 -->
- [ ] Deploy changes to remote server <!-- id: 6 -->
- [ ] Verify the fix on remote server <!-- id: 4 -->
