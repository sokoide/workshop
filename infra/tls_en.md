# TLS/SSL Certificate Workshop: Building Self-Signed CA and Understanding Certificate Chains

In this workshop, you will learn how TLS/SSL and certificate chains work by building your own Certificate Authority (CA) and issuing/verifying server certificates using OpenSSL.

## 1. Basics of TLS/SSL and Certificate Chains

### Why do we need a CA?

To simply encrypt communication on the Internet, you can use a key created by the server itself (self-signed certificate). However, without a third party to prove that the key "really belongs to that server," impersonation cannot be prevented. This "root of trust" is the Certificate Authority (CA).

### Structure of a Certificate Chain

Trust is passed down from top to bottom.

```mermaid
graph TD
    RootCA["Root CA (Self-Signed)"] --> IntermediateCA["Intermediate CA (Optional)"]
    IntermediateCA --> ServerCert["Server Certificate"]
    RootCA -.-> |Import Trust| Browser["OS / Browser Trust Store"]
```

## 2. Preparation

We will use the `openssl` command for this workshop.

```bash
openssl version
```

## 3. Step 1: Building a Root CA

Create a Root CA, which will be the source of all trust.

### 1.1 Generate a private key for the CA

```bash
openssl genrsa -out rootCA.key 4096
```

### 1.2 Create a self-signed root certificate

```bash
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 3650 -out rootCA.crt \
  -subj "/C=JP/ST=Tokyo/L=Minato/O=Workshop/CN=Workshop Root CA"
```

## 4. Step 2: Issuing a Server Certificate

### 2.1 Generate a private key for the server

```bash
openssl genrsa -out server.key 2048
```

### 2.2 Create a Certificate Signing Request (CSR)

```bash
openssl req -new -key server.key -out server.csr \
  -subj "/C=JP/ST=Tokyo/L=Minato/O=Workshop/CN=server.workshop.local"
```

### 2.3 Signing by the CA (Issuing the Certificate)

Modern browsers and tools require **SAN (Subject Alternative Name)** for name validation. Use an extension configuration file to add SAN during signing.

```bash
# Create SAN configuration file
cat <<EOF > server.ext
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = server.workshop.local
DNS.2 = localhost
EOF

# Perform signing
openssl x509 -req -in server.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial \
  -out server.crt -days 365 -sha256 -extfile server.ext
```

## 5. Step 3: Inspecting and Verifying Certificate Content

### 3.1 Inspecting Certificate Details

Display the contents of the issued certificate in text format and verify that the configured items are correctly reflected.

```bash
openssl x509 -noout -text -in server.crt
```

#### How to read the output sample

Information like the following will be displayed:

```text
Certificate:
    Data:
        Version: 3 (0x2)
...
        Issuer: C = JP, ST = Tokyo, L = Minato, O = Workshop, CN = Workshop Root CA
        Validity
            Not Before: Dec 28 10:00:00 2025 GMT
            Not After : Dec 28 10:00:00 2026 GMT
        Subject: C = JP, ST = Tokyo, L = Minato, O = Workshop, CN = server.workshop.local
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                RSA Public-Key: (2048 bit)
        X509v3 extensions:
            X509v3 Basic Constraints: 
                CA:FALSE
            X509v3 Subject Alternative Name: 
                DNS:server.workshop.local, DNS:localhost
...
```

- **Serial Number**: A unique number used to identify certificates issued by the CA.
- **Signature Algorithm**: The algorithm used for signing (sha256 is currently common).
- **Issuer**: The CA that signed this certificate.
- **Validity**: From `Not Before` (start) to `Not After` (end).
- **Subject**: Information about the owner of this certificate.
  - **CN (Common Name)**: Formerly used for identification, but SAN now takes precedence.
- **X509v3 Subject Alternative Name (SAN)**: **The most important item in modern HTTPS**. A list of domain names or IP addresses for which the certificate is valid.

### 3.2 Verifying the Certificate

Verify that the issued certificate is correctly signed by the CA.

```bash
openssl verify -CAfile rootCA.crt server.crt
# Output: server.crt: OK
```

## 6. Step 4: Practical Application with Web Server (Traefik)

Let's use the certificates we created to set up an actual HTTPS server.
We will run a simple Python HTTP server in the background and place **Traefik** in front as an HTTPS (SSL) termination proxy.

### 1. Setting the Hostname

Instead of setting up a DNS server, we will define the domain name in `/etc/hosts`.

```bash
# Append to /etc/hosts
echo "127.0.0.1 server.workshop.local" | sudo tee -a /etc/hosts
```

### 2. Start the Backend Web Server

Use Python to start a simple server running on HTTP (port 8000).

```bash
# Run in a separate terminal
python3 -m http.server 8000
```

### 3. Configure Traefik

Configure Traefik to load the certificates and forward traffic from HTTPS (443) to HTTP (8000).

Create **dynamic_conf.yaml** (Dynamic configuration file).

```yaml
http:
  routers:
    to-python:
      rule: "Host(`server.workshop.local`)"
      service: python-server
      entryPoints:
        - websecure
      tls: {}

  services:
    python-server:
      loadBalancer:
        servers:
          - url: "http://host.containers.internal:8000" # Point to the Python server on the host

tls:
  certificates:
    - certFile: /certs/server.crt
      keyFile: /certs/server.key
```

### 4. Run Traefik Container

```bash
sudo podman run -d --name traefik \
  -p 443:443 \
  --add-host host.containers.internal:host-gateway \
  -v .:/certs:ro \
  -v ./dynamic_conf.yaml:/etc/traefik/dynamic_conf.yaml:ro \
  traefik:v3.1 \
  --providers.file.filename=/etc/traefik/dynamic_conf.yaml \
  --entrypoints.websecure.address=:443
```

## 7. Step 5: Verification from Client (curl)

Access the built HTTPS server using `curl` and see how certificate verification works.

### 1. Access without CA Certificate (Failure case)

Since the Root CA is not in the system's trusted store, verification will fail.

```bash
curl https://server.workshop.local
# Example output: curl: (60) SSL certificate problem: unable to get local issuer certificate
```

### 2. Access with Custom Root CA Certificate (Success case)

By passing our own Root CA certificate with the `--cacert` option, verification succeeds.

```bash
curl --cacert rootCA.crt https://server.workshop.local
# Example output: Success if you see the HTML directory listing from the Python server!
```

## 8. Server Name Validation in HTTPS

When a client such as a browser performs HTTPS communication, it validates whether the connection destination is correct through the following process. This is called **Server Name Validation**.

1. **Verification of Trust**: Verify if the presented certificate is connected from a "trusted root certificate" that the client holds (certificate chain).
2. **Matching the Name**: Match the domain name of the URL entered in the browser with the **SAN (Subject Alternative Name)** of the certificate.
    - **Relationship with CNAME**: CNAME is an alias at the DNS level. In certificate verification, the domain name you are ultimately accessing must be included in the SAN.

### SSL Pass-through and SAN

In an **SSL Pass-through** configuration where SSL termination is not performed at the Load Balancer (LB) and the encrypted data is forwarded as is to the backend Web server, the Web server itself holds the certificate.

In this case, the domain name used in front of the LB (the domain name the client accesses) must match the SAN in the Web server's certificate. If they do not match, the client will display a "security warning."

---

## 9. Cleanup

After the workshop, stop the processes/containers and restore the settings.

```bash
# 1. Stop and remove Traefik container
sudo podman rm -f traefik

# 2. Stop Python server
# Press Ctrl+C in the terminal running the Python server

# 3. Remove generated files (if needed)
rm rootCA.* server.* dynamic_conf.yaml

# 4. Restore /etc/hosts
# Remove the line "127.0.0.1 server.workshop.local"
sudo nano /etc/hosts
```

## 10. Summary

- **Private Key (.key)**: Must never be leaked to the outside.
- **CSR (.csr)**: An "application form" to have the CA issue a certificate.
- **Certificate (.crt)**: The application form stamped (signed) by the CA.
- **Root Certificate**: By installing this into a client (browser, etc.), you can trust all certificates issued by that CA.
