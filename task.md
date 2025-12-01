# Task: Fix High Traffic and Stability Issues

- [x] Implement IP rotation with Reserve Pool
- [x] Fix "Batch TIMED OUT" resource leak
- [x] Add rate limiting to `replenish` function
- [x] Add penalty delay for fast failures in `main.go`
- [x] Optimize timeout recovery (immediate restart)
- [x] Add Discord debug logging
- [x] Fix `slice bounds out of range` panic
- [x] Stagger VLESS adapter startup
- [/] **Cleanup Zombie Processes**
    - [ ] Kill all `xray` and `crawler` processes
    - [ ] Verify no background traffic
- [ ] **Verify Fix**
    - [ ] Run with updated code
    - [ ] Monitor Discord logs for loops
    - [ ] Monitor Cloudflare dashboard
