# Task: Implement Clicking, IP Limits, and Proxy Fallback

- [x] **IP Usage Tracking**
    - [x] Implement robust daily limit (1 run/IP/day)
    - [x] Persist usage data to `ip_usage.json`
- [x] **Proxy Fallback System**
    - [x] Load `proxies.txt` as primary source
    - [x] If all primary IPs used today, fetch free proxies
    - [x] Implement fast availability checker for fetched proxies
    - [ ] **Prevent concurrent fetching (Thundering Herd)** <!-- New -->
- [x] **Browser Automation**
    - [x] Navigate to `https://tsplsimulator.dpdns.org/`
    - [x] Click specific banner (image with `webtrafic.ru`)
    - [x] Close page after click
- [ ] **Configuration**
    - [ ] Ensure `HEADLESS=True` default
    - [ ] Verify 10 threads
- [ ] **Verification**
    - [ ] Verify IP usage limits
    - [ ] Verify banner clicking logic
