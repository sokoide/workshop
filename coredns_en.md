# CoreDNS Workshop: Understanding Name Resolution by Building Parent-Child DNS Servers

This workshop is for software engineers to learn the basic mechanisms of DNS (Authoritative Servers, Forwarding, Delegation) by actually building them using CoreDNS.

## Goal

Build and understand the following configuration **hands-on**.

```
                     DNS Query (test.foo.sokoide.com)
[ Client ] -------------------------------------------------> [ VM1: Parent DNS ]
(Your PC/VM)               (1) Query Port 10053                 | 192.168.100.10
                                                                | Zone: sokoide.com
                                                                |
                                                                | (2) Forward Query
                                                                v
                                                          [ VM2: Child DNS ]
                                                            192.168.100.20
                                                            Zone: foo.sokoide.com
                                                            Returns: 2.2.2.2
```

**What you will learn:**

1. **Authoritative DNS Server:** A server that holds information for its own domain (zone) and answers queries.
2. **Forwarding:** A mechanism to forward queries for unknown domains to another server.
3. **Zone Hierarchy:** The relationship between a parent (`sokoide.com`) and a child (`foo.sokoide.com`).

*Note:* This workshop uses **forwarding** for clarity. Real-world DNS hierarchy is based on **delegation** (NS + glue records), which you will explore in the Next Steps.

---

## Prerequisites

- **2 VMs** (Ubuntu 24.04 recommended)
  - **VM1 (Parent):** IP `192.168.100.10`
  - **VM2 (Child):** IP `192.168.100.20`
  - *Note:* If your IP addresses are different, please replace the IPs in the following steps accordingly.
- **Tools:** `curl`, `tar`, `dig` (dnsutils)

**Preparation (Execute on both VMs):**

```bash
sudo apt update && sudo apt install -y curl tar dnsutils
```

---

## Step 1. Install CoreDNS

CoreDNS is a DNS server written in Go as a single binary. It has no dependencies and is very easy to install.

**Execute on both VM1 and VM2:**

```bash
# Download CoreDNS
CORE_VERSION="1.13.2"
curl -L "https://github.com/coredns/coredns/releases/download/v${CORE_VERSION}/coredns_${CORE_VERSION}_linux_amd64.tgz" -o coredns.tgz

# Extract and place
tar -xzvf coredns.tgz
sudo mv coredns /usr/local/bin/

# Verify operation
coredns -version
```

---

## Step 2. Build VM1 (Parent): sokoide.com

VM1 manages the parent domain `sokoide.com`. Also, configure it to forward queries for the child domain `foo.sokoide.com` to VM2.

**Execute on VM1 (`192.168.100.10`):**

1. Create working directory

   ```bash
   mkdir -p ~/coredns_parent && cd ~/coredns_parent
   ```

2. Create **Corefile** (Configuration file)

   ```bash
   cat <<'EOF' > Corefile
   sokoide.com:10053 {
       file db.sokoide.com
       log
       errors
   }

   foo.sokoide.com:10053 {
       # Forward queries to VM2 (Child)
       forward . 192.168.100.20:10053
       log
       errors
   }
   EOF
   ```

3. Create **Zone file** (Record definitions)

   ```bash
   cat <<'EOF' > db.sokoide.com
   $ORIGIN sokoide.com.
   $TTL 3600
   @   IN  SOA  ns.sokoide.com. root.sokoide.com. (
           2024010101 7200 3600 1209600 3600 )

   @   IN  NS   ns.sokoide.com.
   ns  IN  A    192.168.100.10
   www IN  A    1.1.1.1
   EOF
   ```

---

## Step 3. Build VM2 (Child): foo.sokoide.com

VM2 manages the subdomain `foo.sokoide.com`. Register specific records (e.g., `test`) here.

**Execute on VM2 (`192.168.100.20`):**

1. Create working directory

   ```bash
   mkdir -p ~/coredns_child && cd ~/coredns_child
   ```

2. Create **Corefile**

   ```bash
   cat <<'EOF' > Corefile
   foo.sokoide.com:10053 {
       file db.foo.sokoide.com
       log
       errors
   }
   EOF
   ```

3. Create **Zone file**

   ```bash
   cat <<'EOF' > db.foo.sokoide.com
   $ORIGIN foo.sokoide.com.
   $TTL 3600
   @   IN  SOA  ns1.foo.sokoide.com. root.foo.sokoide.com. (
           2024010101 7200 3600 1209600 3600 )

   @   IN  NS   ns1.foo.sokoide.com.
   ns1  IN  A    192.168.100.20
   test IN  A    2.2.2.2
   EOF
   ```

---

## Step 4. Start DNS Servers

Start CoreDNS on each VM. Keep this terminal open to check logs (or use `screen`/`tmux` or background execution `&`).

**VM1 (Parent) Terminal:**

You can run without `sudo` if `/usr/local/bin` is in your PATH and you are using port `10053` (non-privileged).

```bash
cd ~/coredns_parent
sudo /usr/local/bin/coredns -conf Corefile
```

**VM2 (Child) Terminal:**

```bash
cd ~/coredns_child
sudo /usr/local/bin/coredns -conf Corefile
```

---

## Step 5. Verify Operation (dig)

Verify using the `dig` command from another terminal (or local PC).

### 1. Direct Lookup of Parent Zone

Query `www.sokoide.com` to the parent server (VM1).

```bash
dig @192.168.100.10 -p 10053 www.sokoide.com +short
```

> **Result:** Success if `1.1.1.1` is returned.

### 2. Child Zone Resolution (Forwarding)

Query `test.foo.sokoide.com` (which is in the child zone) to the parent server (VM1).

```bash
dig @192.168.100.10 -p 10053 test.foo.sokoide.com
```

**How to read the output:**

```text
;; ANSWER SECTION:
test.foo.sokoide.com.   3600    IN      A       2.2.2.2  <-- Correct IP (Fetched from VM2)

;; SERVER: 192.168.100.10#10053(192.168.100.10)          <-- Parent server is answering
```

### What happened?

1. The client queried VM1 (`sokoide.com`).
2. VM1 forwarded the query to VM2 according to the configuration (`forward . 192.168.100.20`).
3. VM2 answered `2.2.2.2`.
4. VM1 returned the result to the client.

You can confirm the forwarding and access by checking VM1's logs (depending on log output settings).

---

## Cleanup

Cleanup after the workshop.

1. **Stop processes:** Stop running `coredns` on each VM with `Ctrl+C`.
2. **Delete files:**

   ```bash
   # On both VMs
   rm -rf ~/coredns_parent ~/coredns_child
   # Delete binary if necessary
   sudo rm /usr/local/bin/coredns
   ```

---

## Next Steps

We used **Forwarding** this time, but the basis of DNS on the Internet is **Delegation**.
To perform delegation, write the child's NS record (`foo IN NS ns1.foo...`) and glue record (`ns1.foo IN A ...`) in the parent's zone file (`db.sokoide.com`), allowing the client itself to go query the next server.

Changing the CoreDNS configuration to try out delegation behavior is also good learning.
