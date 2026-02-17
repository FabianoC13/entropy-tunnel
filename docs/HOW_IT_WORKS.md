# How EntropyTunnel Connects to Dublin - Technical Deep Dive

## Overview

```
Your Mac (Spain) ───[Encrypted Tunnel]──→ AWS Server (Dublin) ───→ Internet
     85.54.194.63                              52.48.241.50
```

## The Connection Flow

### 1. Local Setup (Your Mac)

```
Browser/App ──→ SOCKS5 Proxy (127.0.0.1:1080) ──→ entropy-client ──→ xray-core
```

When you enable system-wide VPN:
- macOS routes all HTTP/HTTPS/SOCKS traffic to `127.0.0.1:1080`
- `entropy-client` listens on port 1080
- It forwards everything to `xray-core` (the actual VPN engine)

### 2. The Handshake (TLS 1.3 + Reality)

This is where the magic happens. When xray connects to Dublin:

```
Your Mac                                          AWS Server
   │                                                    │
   │  1. TCP SYN ──────────────────────────────────────→│
   │     (Standard TCP connection to port 443)          │
   │                                                    │
   │  2. TLS ClientHello ──────────────────────────────→│
   │     SNI: www.google.com                            │
   │     Fingerprint: Chrome (uTLS)                     │
   │     Reality Public Key: 8nNZ7Coh...                │
   │                                                    │
   │  3. Reality Handshake ────────────────────────────→│
   │     [Encrypted with X25519 keys]                   │
   │     Looks EXACTLY like Chrome → Google             │
   │                                                    │
   │←────────────────────────────────── 4. TLS ServerHello│
   │     Certificate: Signed by Reality                 │
   │     Looks like Google cert to middleboxes          │
   │                                                    │
   │  5. XTLS-RPRX-VISION handshake ───────────────────→│
   │     [Double encryption layer]                      │
   │                                                    │
   │←═══════════════════════════════════ 6. Tunnel Ready │
   │     🛡️ Encrypted tunnel established!               │
```

### 3. Protocol Stack

```
┌─────────────────────────────────────────────────────────────┐
│  Your Application (Browser, Netflix, etc.)                   │
├─────────────────────────────────────────────────────────────┤
│  HTTP/HTTPS Request                                          │
├─────────────────────────────────────────────────────────────┤
│  SOCKS5 Client (macOS system proxy)                         │
├─────────────────────────────────────────────────────────────┤
│  entropy-client (Go wrapper)                                │
├─────────────────────────────────────────────────────────────┤
│  xray-core VLESS Protocol                                   │
│  - UUID: e9242e9c-6f15-4b49-8d2f-7f1fb4dd1793               │
│  - Flow: xtls-rprx-vision                                   │
├─────────────────────────────────────────────────────────────┤
│  XTLS (Double TLS encryption)                               │
│  - Outer: Reality (camouflage)                              │
│  - Inner: Real payload                                      │
├─────────────────────────────────────────────────────────────┤
│  TCP/443 (HTTPS port)                                       │
├─────────────────────────────────────────────────────────────┤
│  Internet ──→ AWS Dublin ──→ Destination (Netflix, etc.)    │
└─────────────────────────────────────────────────────────────┘
```

## Why This Bypasses Censorship

### 1. **Reality Protocol** - The Invisibility Cloak

```
Normal VPN traffic:
  Client ──[Jibberish packets]──→ Server
  👁️ ISP sees: "This is VPN traffic! BLOCK IT!"

Reality camouflage:
  Client ──[Looks like Chrome→Google]──→ Server
  👁️ ISP sees: "Just someone browsing Google, move along..."
```

The "dest" in config is `www.google.com:443`. This means:
- The TLS handshake mimics a real Chrome browser
- If anyone intercepts and tries to connect, they see real Google
- Your traffic is hidden inside this "decoy" connection

### 2. **uTLS Fingerprinting** - Perfect Chrome Impersonation

```
Real Chrome fingerprint:  
  TLS 1.3, specific cipher suites, extensions order
  
xray with uTLS:
  EXACT same fingerprint as Chrome
  Middlebox fingerprint scanners: "Yep, that's Chrome"
```

### 3. **XTLS-RPRX-VISION** - Double Encryption

```
Standard TLS:  
  [TLS Header][Encrypted Data]
  
XTLS:
  [TLS Header][XTLS Header][Double-Encrypted Data]
  
Even if outer TLS is broken, inner payload is still encrypted
```

## The Actual Bytes on the Wire

### What Your ISP Sees:

```
Packet 1: TCP SYN to 52.48.241.50:443
Packet 2: TLS ClientHello (SNI: www.google.com)
Packet 3: TLS Encrypted Application Data
...
Packet N: More TLS data

Analysis: "Chrome browser connecting to Google. Normal."
```

### What's Actually Inside:

```
[TLS Layer - Looks like Google]
  ↓ decrypt with Reality keys
[Reality Layer - Authenticated]
  ↓ decrypt with session keys  
[VLESS Layer - Your actual VPN traffic]
  ↓ decrypt with UUID
[Your Netflix request]
```

## Server Side (Dublin)

### When packet arrives at 52.48.241.50:

```
AWS Server
   │
   ├── 1. TCP 443 receives packet
   │
   ├── 2. Reality validates:
   │      - Is this a valid client?
   │      - Check public key: 8nNZ7Coh...
   │      - Check short_id: abcdef01
   │      ✓ Valid, proceed
   │
   ├── 3. VLESS authenticates:
   │      - UUID match: e9242e9c...
   │      ✓ Authorized client
   │
   ├── 4. XTLS decrypts inner payload
   │
   └── 5. Forward to destination:
          Your Netflix request ──→ Netflix Ireland servers
          Response ──→ Back through tunnel ──→ Your Mac
```

## Why Netflix Shows Ireland

```
Your Request:
  Your Mac ──→ Dublin Server ──→ Netflix
  
Netflix sees:
  "Request from IP 52.48.241.50 in Dublin, Ireland"
  
Netflix CDN:
  "Route to Ireland content library"
```

## Key Technical Details

| Component | Value | Purpose |
|-----------|-------|---------|
| **Server IP** | 52.48.241.50 | AWS EC2 in eu-west-1 |
| **Port** | 443 | HTTPS (blends with normal traffic) |
| **Protocol** | VLESS | Lightweight proxy protocol |
| **Security** | XTLS-Reality | Camouflaged TLS |
| **Flow** | xtls-rprx-vision | Traffic masking mode |
| **SNI** | www.google.com | Decoy destination |
| **UUID** | e924...1793 | Client authentication |
| **Public Key** | 8nNZ...nHk | Reality handshake |
| **Private Key** | oB5B...02EU | Server-side (secret) |
| **Fingerprint** | chrome | Browser impersonation |

## Traffic Flow Summary

```
┌─────────────┐    ┌──────────────┐    ┌──────────────┐    ┌─────────────┐
│   Netflix   │◄───│ AWS Dublin   │◄───│ Reality/XTLS │◄───│ Your Mac    │
│   Ireland   │    │ 52.48.241.50 │    │   Tunnel     │    │ 85.54.194.63│
└─────────────┘    └──────────────┘    └──────────────┘    └─────────────┘
                                                        ▲
                                                        │
                                              Looks like Google traffic
                                              to anyone watching
```

## Why It's So Fast

1. **XTLS**: Zero overhead encryption (uses hardware AES when possible)
2. **VLESS**: Minimal protocol overhead (no HTTP headers like Shadowsocks)
3. **Reality**: No extra round trips (handshake is standard TLS)
4. **AWS**: 5 Gbps network, low latency Europe-to-Europe

## Security Properties

✅ **Confidentiality**: Double-layer encryption (TLS + VLESS)
✅ **Authentication**: UUID + Reality keys prevent unauthorized access
✅ **Integrity**: All data is cryptographically signed
✅ **Forward Secrecy**: New session keys for each connection
✅ **Censorship Resistance**: Traffic looks identical to Google Chrome

---

*Generated: 2026-02-17*  
*Server: AWS Dublin (52.48.241.50)*
