# Optimizing Proxy Management

- [ ] Analyze `ip_fetcher.go` and `proxy.go` to identify bottlenecks <!-- id: 0 -->
- [ ] Implement IP limit in `FetchPreferredIPs` (e.g., max 50) <!-- id: 1 -->
- [ ] Implement background verification in `MemoryProxyPool` <!-- id: 2 -->
- [ ] Update `RunBatch` to rely on verified proxies (or keep strict check as safety net) <!-- id: 3 -->
- [ ] Verify changes with unit tests <!-- id: 4 -->
- [ ] Deploy and monitor <!-- id: 5 -->
