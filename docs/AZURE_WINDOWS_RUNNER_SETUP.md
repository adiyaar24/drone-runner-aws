# Azure Windows VM Runner Setup Guide

This guide walks you through setting up a Harness CI runner for **Windows VMs on Azure**.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Harness Platform                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Ubuntu VM (Runner Host)                      │
│  ┌─────────────────┐  ┌─────────────────────────────────────┐   │
│  │ Harness Delegate│  │     drone-runner-aws Container      │   │
│  │   (Docker)      │  │  - Manages Windows VM pool          │   │
│  │                 │  │  - Provisions VMs on-demand         │   │
│  └─────────────────┘  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                 Azure Windows VMs (Hot Pool)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ Windows VM  │  │ Windows VM  │  │ Windows VM  │   ...        │
│  │ lite-engine │  │ lite-engine │  │ lite-engine │              │
│  │ Docker      │  │ Docker      │  │ Docker      │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Azure account with VM creation permissions
- Azure Service Principal (client_id, client_secret, tenant_id, subscription_id)
- Ubuntu VM (for running the delegate and runner)
- Network Security Group configured for required ports

## Step 1: Create Ubuntu VM on Azure

Create an Ubuntu VM (20.04 or 22.04 LTS) that will host the Harness Delegate and Runner.

**Recommended specs:**
- Size: Standard_B2s or higher
- OS: Ubuntu 22.04 LTS

## Step 2: Install Docker on Ubuntu VM

SSH into the Ubuntu VM and run:

```bash
# Update system
sudo apt update
sudo apt upgrade -y

# Install prerequisites
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings

# Add Docker GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Enable and start Docker
sudo systemctl enable docker
sudo systemctl start docker

# Add current user to docker group (optional)
sudo usermod -aG docker $USER
newgrp docker
```

## Step 3: Run Harness Delegate

```bash
docker run -d --network host --cpus=1 --memory=2g \
  -e DELEGATE_NAME=<your-delegate-name> \
  -e NEXT_GEN="true" \
  -e DELEGATE_TYPE="DOCKER" \
  -e ACCOUNT_ID=<your-account-id> \
  -e DELEGATE_TOKEN=<your-delegate-token> \
  -e DELEGATE_TAGS="" \
  -e MANAGER_HOST_AND_PORT=https://app.harness.io/gratis \
  us-docker.pkg.dev/gar-prod-setup/harness-public/harness/delegate:26.02.88404
```

Verify delegate is running:
```bash
docker ps
```

Check delegate status in Harness UI.

## Step 4: Configure Runner Pool

Create the runner configuration directory and pool file:

```bash
sudo mkdir -p /runner
cd /runner
sudo nano pool.yml
```

**pool.yml:**
```yaml
version: "1"
instances:
  - name: windows-azure
    type: azure
    pool: 0              # Set to 0 for on-demand, or 1-5 for hot pool
    limit: 5             # Maximum concurrent VMs
    platform:
      os: windows
      arch: amd64
    spec:
      account:
        client_id: <your-client-id>
        client_secret: <your-client-secret>
        subscription_id: <your-subscription-id>
        tenant_id: <your-tenant-id>
      location: westus2
      size: Standard_D4s_v3    # Recommended for faster performance
      resource_group: <your-resource-group>
      security_group_name: <your-nsg-name>
      tags:
        environment: ci
      image:
        username: <vm-username>
        password: <vm-password>
        publisher: MicrosoftWindowsServer
        offer: WindowsServer
        sku: 2022-datacenter-g2
        version: latest
```

### VM Size Recommendations

| Size | vCPUs | RAM | Performance | Use Case |
|------|-------|-----|-------------|----------|
| Standard_B2s | 2 | 4GB | Burstable | Development/Testing |
| Standard_D2s_v3 | 2 | 8GB | Consistent | Light workloads |
| Standard_D4s_v3 | 4 | 16GB | Consistent | **Recommended** |
| Standard_F4s_v2 | 4 | 8GB | Compute-optimized | CPU-intensive builds |

### Pool Settings

| Setting | Description |
|---------|-------------|
| `pool: 0` | On-demand VM creation (waits for full init including Docker restart) |
| `pool: 1-5` | Hot pool with N pre-warmed VMs (faster, but may have stale VMs) |
| `limit` | Maximum number of concurrent VMs |

## Step 5: Run the Runner

```bash
# Login to registry (if using private registry)
echo "<your-token>" | sudo docker login pkg.harness.io -u <your-email> --password-stdin

# Pull the runner image
sudo docker pull pkg.harness.io/<your-org>/aws-runner/drone-runner:windows-fix-amd64

# Run the runner
sudo docker run \
  -v /runner:/runner \
  -p 3000:3000 \
  -e SETUP_TIMEOUT=20m \
  -e HEALTH_CHECK_WINDOWS_TIMEOUT=25m \
  pkg.harness.io/<your-org>/aws-runner/drone-runner:windows-fix-amd64 \
  delegate --pool /runner/pool.yml
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SETUP_TIMEOUT` | 10m | Max time for VM setup |
| `HEALTH_CHECK_WINDOWS_TIMEOUT` | 10m | Max time for Windows health check |

## Step 6: Configure Network Security Groups (NSG)

You need **separate NSG rules** for the Ubuntu VM and Windows VMs.

### Communication Flow

```
┌─────────────────┐         Port 9079          ┌─────────────────┐
│   Ubuntu VM     │ ──────────────────────────▶│   Windows VM    │
│  (Runner)       │                            │  (lite-engine)  │
│                 │◀────────────────────────── │                 │
└─────────────────┘       Response             └─────────────────┘
        │                                              │
        │ Port 443                                     │ Port 443
        ▼                                              ▼
┌─────────────────┐                            ┌─────────────────┐
│ Harness Platform│                            │ Internet        │
│ Docker Registry │                            │ (downloads)     │
└─────────────────┘                            └─────────────────┘
```

---

### Ubuntu VM NSG (Runner Host)

This VM runs the Harness Delegate and drone-runner-aws.

#### Inbound Rules

| Priority | Name | Port | Protocol | Source | Purpose |
|----------|------|------|----------|--------|---------|
| 300 | SSH | 22 | TCP | Your IP | SSH access for management |
| 340 | runner-api | 3000 | TCP | Harness IPs | Harness → Runner (optional) |

#### Outbound Rules

| Priority | Name | Port | Protocol | Destination | Purpose |
|----------|------|------|----------|-------------|---------|
| 100 | harness-api | 443 | TCP | Any | Delegate → Harness Platform |
| 110 | **lite-engine** | **9079** | TCP | Windows VMs / VNet | **Runner → Windows VMs** ⚠️ |
| 120 | docker-registry | 443 | TCP | Any | Pull Docker images |

---

### Windows VMs NSG (CI Execution)

These VMs run lite-engine and execute pipeline steps.

#### Inbound Rules

| Priority | Name | Port | Protocol | Source | Purpose |
|----------|------|------|----------|--------|---------|
| 300 | **lite-engine** | **9079** | TCP | Ubuntu VM IP / VNet | **Runner → lite-engine** ⚠️ REQUIRED |
| 310 | RDP | 3389 | TCP | Your IP | Debugging (optional) |

#### Outbound Rules

| Priority | Name | Port | Protocol | Destination | Purpose |
|----------|------|------|----------|-------------|---------|
| 100 | https | 443 | TCP | Any | Download binaries, images |
| 110 | http | 80 | TCP | Any | Package downloads |

---

### Minimum Required Ports Summary

| From | To | Port | Required | Purpose |
|------|-----|------|----------|---------|
| Ubuntu VM | Windows VM | **9079** | ✅ **YES** | Runner ↔ lite-engine |
| Ubuntu VM | Internet | 443 | ✅ YES | Harness API, Docker registry |
| Windows VM | Internet | 443 | ✅ YES | Download Docker, Git, binaries |
| Your PC | Ubuntu VM | 22 | Optional | SSH management |
| Your PC | Windows VM | 3389 | Optional | RDP debugging |

> **⚠️ Important:** Port **9079** between Ubuntu VM and Windows VMs is **critical** for the runner to communicate with lite-engine.

## Using Private IPs (No Public IP on Windows VMs)

For enhanced security, you can run Windows VMs with **private IPs only**. This requires:

### Prerequisites for Private IP Setup

1. **Same VNet**: Ubuntu VM and Windows VMs must be in the **same Virtual Network** (or peered VNets)
2. **NAT Gateway or Azure Firewall**: Windows VMs need outbound internet access for downloading Docker, Git, etc.
3. **NSG allows VNet traffic**: Ensure NSG allows traffic within the VNet

### Network Architecture (Private IP)

```
┌─────────────────────────────────────────────────────────────────┐
│                     Azure Virtual Network                        │
│                                                                  │
│  ┌──────────────────┐         ┌──────────────────┐              │
│  │   Ubuntu VM      │  9079   │   Windows VM     │              │
│  │   10.0.1.4       │────────▶│   10.0.1.10      │              │
│  │   (Public IP)    │         │   (Private Only) │              │
│  └──────────────────┘         └──────────────────┘              │
│           │                            │                         │
└───────────┼────────────────────────────┼─────────────────────────┘
            │                            │
            ▼                            ▼
    ┌───────────────┐           ┌───────────────┐
    │   Internet    │           │  NAT Gateway  │
    │   (direct)    │           │  (outbound)   │
    └───────────────┘           └───────────────┘
```

### pool.yml Configuration for Private IP

```yaml
version: "1"
instances:
  - name: windows-azure
    type: azure
    pool: 0
    limit: 5
    platform:
      os: windows
      arch: amd64
    spec:
      account:
        client_id: <your-client-id>
        client_secret: <your-client-secret>
        subscription_id: <your-subscription-id>
        tenant_id: <your-tenant-id>
      location: westus2
      size: Standard_D4s_v3
      resource_group: <your-resource-group>
      security_group_name: <your-nsg-name>
      # ▼▼▼ PRIVATE IP SETTINGS ▼▼▼
      network:
        vnet: <your-vnet-name>              # Required for private IP
        subnet: <your-subnet-name>          # Subnet within the VNet
        private_ip: true                    # Use private IP only
      tags:
        environment: ci
      image:
        username: <vm-username>
        password: <vm-password>
        publisher: MicrosoftWindowsServer
        offer: WindowsServer
        sku: 2022-datacenter-g2
        version: latest
```

### NSG Rules for Private IP Setup

#### Ubuntu VM NSG

| Direction | Port | Protocol | Source/Dest | Purpose |
|-----------|------|----------|-------------|---------|
| Outbound | 9079 | TCP | **VirtualNetwork** | Runner → Windows VMs |
| Outbound | 443 | TCP | Any | Harness API |

#### Windows VMs NSG

| Direction | Port | Protocol | Source/Dest | Purpose |
|-----------|------|----------|-------------|---------|
| Inbound | 9079 | TCP | **VirtualNetwork** | lite-engine from Runner |
| Outbound | 443 | TCP | Any | Downloads (via NAT) |
| Outbound | 80 | TCP | Any | Downloads (via NAT) |

### Setting Up NAT Gateway for Windows VMs

Windows VMs need outbound internet for downloading Docker, Git, and binaries. Configure a NAT Gateway:

```bash
# Create NAT Gateway public IP
az network public-ip create \
  --resource-group <your-rg> \
  --name nat-gateway-ip \
  --sku Standard

# Create NAT Gateway
az network nat gateway create \
  --resource-group <your-rg> \
  --name windows-nat-gateway \
  --public-ip-addresses nat-gateway-ip \
  --idle-timeout 10

# Associate with subnet
az network vnet subnet update \
  --resource-group <your-rg> \
  --vnet-name <your-vnet> \
  --name <your-subnet> \
  --nat-gateway windows-nat-gateway
```

### Comparison: Public IP vs Private IP

| Aspect | Public IP | Private IP |
|--------|-----------|------------|
| Security | Less secure (exposed) | More secure (isolated) |
| Setup complexity | Simple | Requires VNet + NAT |
| Cost | Public IP cost | NAT Gateway cost |
| Debugging | Easy (RDP direct) | Harder (need bastion/jump) |
| Recommended for | Development | Production |

---

## Windows VM Initialization

The runner automatically initializes Windows VMs with:

1. **Containers Feature** - Required for Docker
2. **Docker CE** - Container runtime
3. **Git** - For repository operations
4. **lite-engine** - Harness CI execution agent
5. **TLS Certificates** - Secure communication

### Initialization Timeline

| Phase | Duration | Actions |
|-------|----------|---------|
| Phase 1 | ~3-5 min | Install features, Docker, Git, certs, download binaries |
| Restart | ~2 min | VM reboot for Containers feature |
| Phase 2 | ~1-2 min | Start Docker service, start lite-engine |
| **Total** | **~6-10 min** | Full initialization |

> **Note:** With `pool: 0`, the runner waits for full initialization. With hot pool (`pool: 1+`), VMs are pre-warmed but may require restart cycle time.

## Troubleshooting

### Check Windows VM Status

```powershell
# Check lite-engine
Get-Process -Name "lite-engine"
netstat -an | findstr "9079"

# Check Docker
Get-Service docker
docker version

# Check logs
Get-Content "C:\Program Files\lite-engine\log.err" -Tail 50
Get-Content "C:\Program Files\lite-engine\phase2.log"
```

### Check Custom Script Extension Status

```powershell
Get-ChildItem "C:\Packages\Plugins\Microsoft.Compute.CustomScriptExtension\*\Status\" -Recurse | Get-Content
```

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `context deadline exceeded` | VM still initializing | Increase `HEALTH_CHECK_WINDOWS_TIMEOUT` |
| `docker-users not found` | Docker config issue | Script auto-creates group now |
| `failed to find stage in store` | Stale database entries | Clean `/runner/` directory |
| Docker won't start | Containers feature needs restart | Wait for Phase 2 to complete |

### Clean Runner State

```bash
sudo docker stop $(sudo docker ps -q)
cd /runner
sudo rm -rf runner/ *.db
# Restart runner
```

## License

Apache 2.0
