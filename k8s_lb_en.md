# K8s Service (LoadBalancer) Workshop: Building a Virtual LB with iptables

This workshop extends the environment from [vlan_en.md](./vlan_en.md).
We will learn how Kubernetes `Service (Type: LoadBalancer)` and `MetalLB` manipulate packets behind the scenes by reproducing the behavior using standard Linux features.

## Goal

**"Access a backend server (isolated) via a frontend Representative IP (VIP)."**

```mermaid
graph LR
    subgraph "VLAN 20 (Client Side)"
        Client[Container B<br>192.168.20.20]
        VIP[Router VIP<br>192.168.20.100]
    end

    subgraph "Router Container"
        Firewall["iptables: DROP<br>(Block Direct Access)"]
        DNAT["iptables: DNAT<br>(VIP:80 -> Pod:80)"]
    end

    subgraph "VLAN 10 (Cluster/Pod Side)"
        Pod[Container A<br>192.168.10.10]
    end

    Client -- "1. curl 192.168.20.100" --> VIP
    VIP -- "2. DNAT" --> DNAT
    DNAT -- "3. Forward" --> Pod
    Client -. "x Direct Ping x" .-x Pod
```

## Prerequisites

- Complete up to Step 6 of `vlan_en.md`. Containers `a`, `b`, and `router` must be running.

---

## Step 1. Prepare the Web Server

Start a simple web server on the backend server (Container A).

```bash
# Install netcat if missing
sudo podman exec a apk add --no-cache busybox-extras

# Start a simple HTTP server on Container A (Port 80)
sudo podman exec -d a sh -c "while true; do echo -e 'HTTP/1.1 200 OK\n\nHello from Pod A' | nc -l -p 80; done"
```

---

## Step 2. Network Isolation (Firewall)

For this experiment, we block direct communication from VLAN 20 (Client) to VLAN 10 (Server).
In K8s, it is a principle to "use Service IP instead of accessing Pod IP directly." We will forcibly reproduce this.

```bash
# Add a rule to block forwarding on the Router
sudo podman exec router iptables -I FORWARD -i eth1 -o eth0 -j DROP
```

**Verification:**
Confirm that Ping from Container B to A fails.

```bash
sudo podman exec b ping -c 2 192.168.10.10
# Result: 100% packet loss (Unreachable)
```

---

## Step 3. Configure VIP (Virtual IP)

Add a new IP address (`192.168.20.100`) to the router's VLAN 20 interface (`eth1`).
This corresponds to the **LoadBalancer IP (External IP)** in K8s.

```bash
# Add IP to eth1
sudo podman exec router ip addr add 192.168.20.100/32 dev eth1
```

- **Note:** `/32` means a single host IP (just this VIP). MetalLB (L2 mode) does essentially the same thing. For ARP requests ("Who has this IP?"), the router answers "I have it (here is the MAC address)."

---

## Step 4. Configure DNAT (Destination NAT)

This is the **core of K8s Services**.
Write a rule to "Forward access destined for VIP (`192.168.20.100`) to Pod A (`192.168.10.10`)."

```bash
# DNAT Rule: If destination is 20.100:80, rewrite to 10.10:80
sudo podman exec router iptables -t nat -A PREROUTING \
  -d 192.168.20.100 -p tcp --dport 80 \
  -j DNAT --to-destination 192.168.10.10:80
```

---

## Step 5. Handle Return Traffic (SNAT)

Since we blocked "Direct Communication" in Step 2, the reply from Pod A cannot return to Client B (or routing inconsistencies will occur).
Therefore, we add a setting (SNAT) to "make it appear to Pod A that the request came from the Router (`192.168.10.1`)."

```bash
# SNAT (Masquerade) Rule
# If destination is 10.10:80, rewrite source to Router's IP
sudo podman exec router iptables -t nat -A POSTROUTING \
  -d 192.168.10.10 -p tcp --dport 80 \
  -j MASQUERADE
```

- **Note:** In K8s, `kube-proxy` configures these automatically. Also, when using Cloud LBs, similar translation occurs when forwarding to `NodePort`.

---

## Step 6. Verify Operation

Access the VIP (`192.168.20.100`) from Container B.

```bash
# Install curl (if not installed)
sudo podman exec b apk add --no-cache curl

# Access VIP
sudo podman exec b curl http://192.168.20.100
```

**Expected Result:**

```text
Hello from Pod A
```

We confirmed that while direct access to Container A's IP (`10.10`) is blocked, access is possible via the VIP (`20.100`). This is the fundamental behavior of a Load Balancer.

**Cleanup (optional):** remove the temporary firewall rule after the experiment.

```bash
sudo podman exec router iptables -D FORWARD -i eth1 -o eth0 -j DROP
```

---

## Explanation: Mapping to K8s

| This Workshop | K8s Component / Setting |
| :--- | :--- |
| `ip addr add 192.168.20.100` | **MetalLB (Speaker)** <br> The elected leader node advertises the IP. |
| `iptables ... -j DNAT` | **kube-proxy** <br> Translates access to Service IP (ClusterIP/NodePort) to Pod IP. |
| `iptables ... -j DROP` | **NetworkPolicy** (Deny All) <br> Blocks unnecessary direct communication. |

Through this workshop, you should understand that a Load Balancer is not "magic," but a combination of **"IP Address Advertisement"** and **"Packet Destination Rewriting (NAT)."**

```
