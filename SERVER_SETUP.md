# Server Setup Guide

## AWS EC2 Instance (Ireland)

Instance: i-03335fe35b296376c
IP: 52.48.241.50
Region: eu-west-1 (Ireland)
Type: t3.micro

## Xray Keys

- UUID: e9242e9c-6f15-4b49-8d2f-7f1fb4dd1793
- Private Key (server): oB5BNRSZKZ7nR-LNO-6gprEUoAqxTUFeP6vauEm02EU
- Public Key (client): 8nNZ7Coh5u3ILM_SuUKW-Sp6daaOFSLYxrpxJLISnHk
- Short ID: abcdef01

## Connection Test

```bash
# Direct connection (shows Spanish IP)
curl ifconfig.me

# Through VPN (shows Irish IP)
curl --socks5-hostname 127.0.0.1:1080 ifconfig.me
```

## Build with xray exec support

```bash
make client-exec
./bin/entropy-client connect -c configs/client.yaml
```
