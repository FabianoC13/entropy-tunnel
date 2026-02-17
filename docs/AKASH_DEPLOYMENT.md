# EntropyTunnel Akash Deployment Guide

## Quick Start

### 1. Set Your API Key

```bash
export AKASH_API_KEY="ac.sk.production.f27d65a063759be56d5ff429868ce99f1dc06f20e2cc520a76bd2b54d9aa1f08"
```

### 2. Deploy to Akash

```bash
# Make script executable
chmod +x scripts/akash/deploy-akash.sh

# Create deployment
./scripts/akash/deploy-akash.sh create
```

This will:
1. Upload the SDL to Akash Network
2. Wait for a provider to accept the lease (2-5 minutes)
3. Output the server URI for your client config

### 3. Update Client Config

Once deployment is ready, update `configs/client.yaml`:

```yaml
server: "YOUR_AKASH_URI:443"
uuid: "RETRIEVED_FROM_LOGS"
public_key: "RETRIEVED_FROM_LOGS"
short_id: "RETRIEVED_FROM_LOGS"
```

### 4. Connect

```bash
./bin/entropy-client connect -c configs/client.yaml
```

---

## Alternative: Direct API Deployment

If the script doesn't work, use curl directly:

```bash
# 1. Read SDL
SDL=$(cat deployments/akash/xray-server.yaml | jq -Rs .)

# 2. Create deployment
 curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AKASH_API_KEY" \
  -d "{\"sdl\": $SDL}" \
  https://api.cloudmos.io/v1/deployments

# 3. Check status (replace DSEQ with returned value)
curl -H "Authorization: Bearer $AKASH_API_KEY" \
  https://api.cloudmos.io/v1/deployments/YOUR_DSEQ
```

---

## Retrieving Credentials

The container auto-generates Xray keys on startup. To get them:

### Option 1: Cloudmos Web Console
1. Go to https://deploy.cloudmos.io
2. Log in with your API key
3. Find your deployment
4. Click "Logs" to see the credentials output

### Option 2: Akash CLI (if installed)
```bash
akash provider lease-logs --dseq YOUR_DSEQ --provider PROVIDER_ADDRESS
```

Look for lines starting with:
```
UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
PUBLIC_KEY: xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SHORT_ID: xxxxxxxx
HOSTNAME: xxx.xxx.xxx.xxx
```

---

## Cost

Akash deployments are typically:
- **~$5-10/month** for small instances (0.5 CPU, 512MB RAM)
- Billed in $AKT (Akash token)
- Pay-as-you-go, can close anytime

Current pricing in SDL:
- 0.5 vCPU
- 512MB RAM
- 1GB storage
- ~10,000 uakt/hour (~$0.01/hour)

---

## Managing Deployments

### Check Status
```bash
./scripts/akash/deploy-akash.sh status
```

### View Logs
```bash
./scripts/akash/deploy-akash.sh logs
```

### Close Deployment
```bash
./scripts/akash/deploy-akash.sh close
```

### List All Deployments
```bash
./scripts/akash/deploy-akash.sh list
```

---

## Integration with Rotation Controller

The Akash controller is integrated with the rotation system:

```go
// Create Akash controller
config := akash.Config{
    APIKey:  os.Getenv("AKASH_API_KEY"),
    SDLPath: "deployments/akash/xray-server.yaml",
}

controller, err := akash.NewController(config, logger)
if err != nil {
    log.Fatal(err)
}

// Rotate to new Akash deployment
endpoint, err := controller.RotateToAkash(context.Background())

// Or use with rotation manager
manager := rotation.NewManager(controller, logger)
manager.StartAutoRotation(context.Background(), 24*time.Hour)
```

---

## Troubleshooting

### Deployment stuck in "pending"
- Akash providers may take 2-5 minutes to bid
- Check Cloudmos console for bid status
- Ensure SDL pricing is competitive

### Can't connect after deployment
- Verify credentials were retrieved from logs
- Check firewall rules (port 443 must be open)
- Try `curl https://YOUR_URI:443` to test

### API errors
- Verify API key is valid: starts with `ac.sk.`
- Check API status: https://status.cloudmos.io

---

## Security Notes

- Keys are generated fresh in each container
- No persistent storage (except logs)
- Decentralized providers = resistant to shutdown
- Can rotate to new provider anytime

---

## Links

- Akash Network: https://akash.network
- Cloudmos Console: https://deploy.cloudmos.io
- Documentation: https://docs.akash.network
