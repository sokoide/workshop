# TLS/SSL 証明書実習：自己署名 CA 構築と証明書チェーン

このワークショップでは、OpenSSL を使用して独自の認証局 (CA) を構築し、サーバー証明書を発行・検証するプロセスを通じて、TLS/SSL および証明書チェーンの仕組みを学びます。

> **💡 用語集**: この実習で登場する専門用語は [用語集](glossary.md) を参照してください。

## ゴール

独自のルート CA を構築し、SAN (Subject Alternative Name) に対応したサーバー証明書を発行して、Traefik による HTTPS 終端を構成します。

```mermaid
graph TD
    RootCA["Root CA (自己署名)"] --> ServerCert["Server Certificate"]
    RootCA -.-> |信頼のインポート| Client["curl (cacert)"]
    ServerCert --> Traefik["Traefik (SSL 終端)"]
    Traefik --> App["Backend App (HTTP)"]
```text

**この実習で習得すること:**

1. **認証局 (CA) の役割**: なぜ信頼の起点が必要なのかの理解。
2. **証明書発行フロー**: 秘密鍵生成、CSR 作成、CA による署名の流れ。
3. **モダンな名前検証**: Common Name (CN) から Subject Alternative Name (SAN) への移行。
4. **証明書の検証**: `curl` や `openssl verify` を使った正当性の確認。

---

## 証明書の信頼における課題

インターネット上の通信を暗号化するだけなら、サーバーが自分で作った鍵（自己署名証明書）でも可能です。

### ❌ 課題

- **なりすまし防止不可**: その鍵が「本当にそのサーバーのものか」を証明する第三者がいない。
- **ブラウザの警告**: 信頼されていない CA から発行された証明書は、安全でないと判断される。

### ✅ 認証局 (CA) の解決策

- **信頼の起点**: 全てのブラウザ/OS があらかじめ信頼している「ルート CA」が存在し、そこから署名（ハンコ）を貰うことで、正当性を証明します。
- **証明書チェーン**: ルート CA が中間 CA を保証し、中間 CA がサーバーを保証する「連鎖」により信頼を構築します。

---

## アーキテクチャ

本実習ではローカル環境で全てのロール（CA、サーバー、クライアント）を再現します。

### 想定ディレクトリ構造

```text
~/
├── rootCA.key        # CA の秘密鍵 (最重要)
├── rootCA.crt        # CA の証明書 (公開用)
├── server.key        # サーバーの秘密鍵
├── server.csr        # サーバーの署名要求
├── server.crt        # サーバーの証明書 (CA署名済)
└── dynamic_conf.yaml # Traefik の TLS 設定
```text

---

## 実習ステップ

### ✅ チェックポイント

- [ ] `openssl version` で OpenSSL がインストールされていることを確認した
- [ ] `podman version` または `docker version` でコンテナ実行環境があることを確認した

### STEP 1: ルート CA (Root CA) の構築

すべての信頼の源となる「自分専用の認証局」を作成します。

```bash
# 1.1 CA 用の秘密鍵を生成
openssl genrsa -out rootCA.key 4096

# 1.2 自己署名ルート証明書を作成 (有効期限10年)
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 3650 -out rootCA.crt \
  -subj "/C=JP/ST=Tokyo/L=Minato/O=Workshop/CN=Workshop Root CA"
```text

### STEP 2: サーバー証明書の発行

次に、実際の Web サーバー（Traefik）で使用する証明書を作成し、CA で署名します。

```bash
# 2.1 サーバー用の秘密鍵を生成
openssl genrsa -out server.key 2048

# 2.2 証明書署名要求 (CSR) の作成
openssl req -new -key server.key -out server.csr \
  -subj "/C=JP/ST=Tokyo/L=Minato/O=Workshop/CN=server.workshop.local"

# 2.3 SAN 設定ファイルの作成 (現代のブラウザでは必須)
cat <<EOF > server.ext
subjectAltName = @alt_names
[alt_names]
DNS.1 = server.workshop.local
DNS.2 = localhost
EOF

# 2.4 CA による署名
openssl x509 -req -in server.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial \
  -out server.crt -days 365 -sha256 -extfile server.ext
```text

### ✅ チェックポイント

- [ ] `ls -la server.crt` で証明書ファイルが生成されていることを確認した
- [ ] `openssl x509 -in server.crt -noout -text` で `Subject Alternative Name` に `server.workshop.local` が含まれていることを確認した

### STEP 3: 証明書の検証

```bash
openssl verify -CAfile rootCA.crt server.crt
# -> server.crt: OK と出れば成功
```text

### STEP 4: Web サーバーでの実践利用 (Traefik)

作成した証明書を使用して、実際に HTTPS サーバーを構成します。

```bash
# 1. ドメイン名のローカル解決設定
echo "127.0.0.1 server.workshop.local" | sudo tee -a /etc/hosts

# 2. バックエンド Web サーバーの起動 (Python)
python3 -m http.server 8000 &

# 3. Traefik 起動 (証明書をマウント)
sudo podman run -d --name traefik -p 443:443 \
  -v .:/certs:ro \
  -v ./dynamic_conf.yaml:/etc/traefik/dynamic_conf.yaml:ro \
  docker.io/library/traefik:v3.1 \
  --providers.file.filename=/etc/traefik/dynamic_conf.yaml \
  --entrypoints.websecure.address=:443
```text

### STEP 5: クライアントからの検証 (curl)

```bash
# CA 証明書を指定せずにアクセス (失敗するはず)
curl https://server.workshop.local

# 自作のルート CA 証明書を指定してアクセス (成功するはず)
curl --cacert rootCA.crt https://server.workshop.local
```text

---

## Clean Architecture と SSL 終端

本実習では **SSL 終端 (SSL Termination)** という構成を取りました。

- **Traefik (Infrastructure Layer)**: SSL 終端を担当。証明書の管理や暗号化・復号の複雑さを一手に引き受けます。
- **App (Business Logic Layer)**: 背後の Python アプリは SSL を意識せず、純粋な HTTP で動作します。

これにより、アプリケーションコードを HTTPS 対応のために変更することなく、インフラ層の差し替えだけでセキュリティを担保できます。

---

## 片付け

```bash
sudo podman rm -f traefik
kill %1  # Python サーバーの停止
rm rootCA.* server.* dynamic_conf.yaml
```text

---

## 参考文献

- [OpenSSL Documentation](https://www.openssl.org/docs/)
- [Traefik TLS Documentation](https://doc.traefik.io/traefik/https/tls/)

---

## トラブルシューティング

### curl で証明書エラーが発生する

**症状**: `curl: (60) SSL certificate problem: unable to get local issuer certificate`

**原因と対処**:

- CA 証明書が正しく指定されているか確認

    ```bash
    curl --cacert rootCA.crt https://server.workshop.local
    ```

- サーバー証明書が CA で署名されているか確認

    ```bash
    openssl verify -CAfile rootCA.crt server.crt
    # server.crt: OK と表示されるはず
    ```

### Traefik が起動しない

**症状**: Traefik コンテナがすぐに終了する

**原因と対処**:

- 証明書ファイルのパスが正しいか確認

    ```bash
    ls -la rootCA.crt server.crt server.key
    ```

- dynamic_conf.yaml の構文を確認

    ```bash
    cat dynamic_conf.yaml
    ```

- Traefik のログを確認

    ```bash
    sudo podman logs traefik
    ```

### 証明書の検証に失敗する

**症状**: `openssl verify` でエラー

**原因と対処**:

- SAN (Subject Alternative Name) が正しく設定されているか確認

    ```bash
    openssl x509 -in server.crt -noout -text | grep -A1 "Subject Alternative Name"
    ```

- ドメイン名が一致しているか確認

    ```bash
    cat /etc/hosts | grep server.workshop.local
    ```

---

## 環境別注意事項

### macOS の場合

OpenSSL コマンドは同じですが、Traefik を Podman で動かす場合は VM 環境が必要です。

- Docker Desktop for Mac を使用する場合、ポートフォワーディング設定を確認
- `brew install openssl` で最新版の OpenSSL をインストール可能

### Windows の場合

- WSL2 上の Ubuntu で実行することを推奨
- PowerShell ではなく WSL2 のターミナルでコマンドを実行
