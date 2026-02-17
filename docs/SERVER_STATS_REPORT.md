# EntropyTunnel Server - Full Stats Report
Generated: 2026-02-17 18:57 UTC

## 1. CURRENT SERVER STATUS

### Hardware Specs (t3.micro)
```
Provider:        AWS EC2
Region:          eu-west-1 (Ireland)
AZ:              - (metadata blocked)
Instance Type:   t3.micro
vCPUs:           2 (Intel Xeon Platinum 8175M @ 2.50GHz)
Memory:          916 MB (261 MB used, 522 MB available)
Disk:            30 GB NVMe (3.6 GB used, 27 GB free)
Uptime:          10 minutes
Load Average:    0.07 0.17 0.10
```

### Xray Process Status
```
PID:             4806
User:            nobody
Memory:          9.4 MB RSS
CPU Time:        161ms
Status:          Active (running since 17:48 UTC)
Listen Port:     443 (VLESS + Reality)
Connections:     0 active (currently idle)
Service:         systemd enabled
```

### Network Status
```
Public IP:       52.48.241.50
Listening:       :443 (xray), :22 (SSH)
RX Bytes:        84.9 MB (63K packets)
TX Bytes:        901 KB (7.5K packets)
Firewall:        AWS Security Group + iptables
```

---

## 2. CURRENT LIMITS & BOTTLENECKS

### System Limits
| Resource | Current Limit | Bottleneck? |
|----------|---------------|-------------|
| Open Files | 65,535 | ✅ Good |
| Max Connections (somaxconn) | 4,096 | ⚠️ Moderate |
| TCP SYN Backlog | 128 | 🔴 LOW - Will drop under load |
| Local Port Range | 32,768-60,999 | ✅ Good (~28K ports) |
| User Processes | Unlimited | ✅ Good |
| File Locks | Unlimited | ✅ Good |

### Network Limits
| Metric | t3.micro Limit | Notes |
|--------|----------------|-------|
| Baseline Bandwidth | 5 Gbps | Burstable |
| Baseline CPU Credits | 12/hour | Can burst to 100% |
| Max Throughput | ~625 MB/s | Theoretical |
| Practical VPN Users | 50-100 | With encryption overhead |

### Security Status
```
✅ No failed login attempts (btmp clean)
✅ xray runs as unprivileged user (nobody)
✅ Firewall rules active
⚠️ No fail2ban installed
⚠️ SSH on port 22 (could move)
⚠️ TCP SYN backlog low (128)
```

---

## 3. CAPACITY ANALYSIS

### Single Server Capacity (t3.micro)
```
Concurrent Users:     ~50-100 users
Throughput:           ~500 Mbps peak
Memory per user:      ~2-3 MB
CPU per user:         Low (Go is efficient)

Bottleneck will be:   CPU credits under sustained load
                      Network bandwidth with many users
```

### When to Scale Up
| Users | Action | Instance |
|-------|--------|----------|
| 0-50 | Current setup | t3.micro ✅ |
| 50-200 | Upgrade | t3.small ($17/mo) |
| 200-500 | Upgrade | t3.medium ($34/mo) |
| 500+ | Multiple servers | Load balancer + 2+ instances |

---

## 4. SCALABILITY RECOMMENDATIONS

### Phase 1: Optimize Current Server (FREE)
1. **Increase TCP SYN backlog**
   ```bash
   sudo sysctl -w net.ipv4.tcp_max_syn_backlog=65535
   sudo sysctl -w net.core.netdev_max_backlog=65535
   ```

2. **Enable BBR congestion control**
   ```bash
   sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
   ```

3. **Increase file descriptors for xray**
   Already at 65,535 ✅

4. **Enable connection tracking optimization**
   ```bash
   sudo sysctl -w net.netfilter.nf_conntrack_max=1000000
   ```

### Phase 2: Horizontal Scaling (Paid)
```
Architecture: Multi-region deployment

              [ User ]
                 |
           [ DNS/GeoDNS ]
            /          \
    [ EU-West-1 ]   [ EU-Central-1 ]
    (Ireland)         (Frankfurt)
         |                  |
    [ t3.micro ]      [ t3.micro ]
    52.48.241.50      (new instance)
         \                  /
          [   Monitoring   ]

Cost: ~$20-30/month for 2 regions
```

### Phase 3: High Availability (Enterprise)
```
              [ CloudFlare / DNS ]
                      |
              [ AWS NLB / ALB ]
                 /        \
        [ t3.medium ]  [ t3.medium ]
        (Primary)       (Failover)
             \            /
          [ Shared nothing ]

Cost: ~$80-150/month
```

---

## 5. LONG-TERM MAINTENANCE NEEDS

### Monitoring Stack (Install)
```yaml
Prometheus:     Metrics collection
Grafana:        Visualization  
Node Exporter:  System metrics
Xray Exporter:  VPN metrics (custom)
AlertManager:   Notifications

Cost: Free (open source)
Setup: 2-3 hours
```

### Log Management
```yaml
Current:        Local files (/var/log/xray/)
Recommended:    AWS CloudWatch Logs
Alternative:    Loki + Grafana
Retention:      30 days (compliance)

Cost: ~$5-10/month for logs
```

### Backup Strategy
```yaml
Critical Data:
  - Xray config (/usr/local/etc/xray/config.json)
  - SSL/TLS certificates (if any)
  - User credentials (UUIDs, keys)

Backup Method:
  - Git repo (config only)
  - S3 bucket (encrypted)
  - Frequency: On change

Recovery Time: < 5 minutes
```

### Update Schedule
```
Daily:    Security patches (automatic)
Weekly:   xray-core version check
Monthly:  OS updates review
Quarterly: Disaster recovery test
```

---

## 6. COST PROJECTIONS

### Current (1 Server, t3.micro)
| Component | Monthly Cost |
|-----------|--------------|
| EC2 Instance | $9.12 |
| Data Transfer (100GB) | $9.00 |
| EBS Storage | $2.40 |
| **TOTAL** | **~$20/month** |

### Growth Projections
| Users | Setup | Monthly Cost |
|-------|-------|--------------|
| 1-50 | 1x t3.micro | $20 |
| 50-200 | 1x t3.small | $35 |
| 200-500 | 1x t3.medium | $60 |
| 500-2000 | 2x t3.medium + LB | $140 |
| 2000+ | Auto-scaling group | $200+ |

### Free Tier Optimization
```
If using AWS Free Tier (12 months):
- 750 hours EC2/month: FREE
- 100GB data transfer: FREE
- 30GB EBS: FREE
- ACTUAL COST: $0/month for year 1
```

---

## 7. SECURITY HARDENING CHECKLIST

### Immediate (Do Now)
- [ ] Change SSH port from 22 to non-standard
- [ ] Install fail2ban for SSH brute force protection
- [ ] Disable root login via SSH
- [ ] Add SSH key only (disable password auth)

### Short Term (This Week)
- [ ] Enable AWS CloudTrail
- [ ] Setup VPC Flow Logs
- [ ] Configure AWS Config for compliance
- [ ] Enable automatic security updates

### Long Term (This Month)
- [ ] Implement mTLS for xray connections
- [ ] Setup log aggregation and alerting
- [ ] Regular penetration testing
- [ ] Compliance audit (if handling user data)

---

## 8. PERFORMANCE TUNING COMMANDS

### Apply These Now
```bash
# SSH into server and run:

# 1. Increase connection limits
sudo tee -a /etc/sysctl.conf << 'EOF'
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_notsent_lowat = 16384
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
EOF

# 2. Apply changes
sudo sysctl -p

# 3. Tune xray service limits
sudo mkdir -p /etc/systemd/system/xray.service.d/
sudo tee /etc/systemd/system/xray.service.d/limits.conf << 'EOF'
[Service]
LimitNOFILE=65535
LimitNPROC=65535
EOF

sudo systemctl daemon-reload
sudo systemctl restart xray
```

---

## 9. SUMMARY & NEXT STEPS

### Current Status: ✅ OPERATIONAL
- Server: Healthy, low load
- VPN: Functional, tested
- Security: Basic, needs hardening
- Capacity: 50-100 users max

### Priority Actions
1. **HIGH**: Apply kernel tuning (5 min)
2. **HIGH**: Install fail2ban (10 min)
3. **MED**: Setup CloudWatch monitoring (30 min)
4. **MED**: Create automated backups (1 hour)
5. **LOW**: Document user management (ongoing)

### Estimated Time to Production-Ready
- Basic stability: **NOW** ✅
- Production hardened: **4-6 hours**
- Enterprise grade: **2-3 days**

---

*Report generated by Patito 🦆*
*Server: 52.48.241.50 (AWS EC2 t3.micro, Ireland)*
