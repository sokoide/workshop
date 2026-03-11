# Kerberos / SPNEGO 認証実習：NGINX と Podman でシングルサインオンを体験する

ソフトウェアエンジニア向けに、エンタープライズ環境で標準的に利用されている **Kerberos** および **SPNEGO** 認証の仕組みを、**NGINX** と **Podman** を使って実際に構築しながら学びます。

> **💡 用語集**: この実習で登場する[Kerberos](glossary_ja.md#network)や[SPN](glossary_ja.md#network)などの専門用語は [用語集](glossary_ja.md) を参照してください。

## ゴール

Podman 上に KDC（Key Distribution Center）と NGINX サーバーを構築し、ブラウザ（または curl）からパスワード入力なしで認証が成功する「シングルサインオン」の流れを理解します。

```mermaid
sequenceDiagram
    participant C as クライアント (Host)
    participant K as KDC (kdc.test.local)
    participant N as NGINX (nginx.test.local)

    Note over C,K: (1) kinit (TGT取得)
    C->>K: AS-REQ (User ID)
    K-->>C: AS-REP (TGT)

    Note over C,K: (2) HTTP サービスチケット要求
    C->>K: TGS-REQ (TGT + SPN)
    K-->>C: TGS-REP (Service Ticket)

    Note over C,N: (3) SPNEGO 認証
    C->>N: HTTP GET (Authorization: Negotiate <Ticket>)
    N-->>C: 200 OK (User: user1@TEST.LOCAL)
```

**この実習で習得すること:**

1. **Kerberos の基本コンポーネント:** KDC、プリンシパル、Keytab の役割。
2. **SPN (Service Principal Name):** サービスを識別するための名前解決と認証の紐付け。
3. **SPNEGO (Negotiate):** HTTP プロトコル上で Kerberos 認証を行う仕組み。
4. **NGINX へのモジュール組み込み:** 動的モジュールを用いた認証機能の拡張。

---

## なぜ Kerberos / SPNEGO なのか？

Active Directory 環境などの社内ネットワークにおいて、一度のログインで複数の Web サービスに自動ログインできる仕組みは、セキュリティと利便性の両立に不可欠です。

- **パスワードをネットワークに流さない**: チケットベースの認証により、認証情報の漏洩リスクを低減します。
- **相互認証**: クライアントだけでなく、サーバーが本物であることも確認できます。
- **標準プロトコル**: Windows, Linux, macOS などの異なる OS 間で共通の認証基盤を構築できます。

---

## アーキテクチャ

Podman-compose を利用し、KDC と NGINX の 2 つのコンテナを同一ネットワーク内で実行します。

### ディレクトリ構造

```text
kerberos_lab/
├── docker-compose.yml
├── Dockerfile.kdc
├── Dockerfile.nginx
├── krb5.conf          # 共通の Kerberos 設定
├── nginx.conf         # SPNEGO 設定を含む NGINX 設定
└── (krb5.keytab)      # STEP 2 で生成される
```

---

## 準備

### 1. ツールインストール (ホストマシン)

```bash
# Ubuntu/Debian の場合
sudo apt update && sudo apt install -y krb5-user curl podman podman-compose
```

### 2. ホスト解決の設定

コンテナの名前解決ができるよう、ホストマシンの `/etc/hosts` に追記します。

```bash
sudo sh -c 'echo "127.0.0.1 kdc.test.local nginx.test.local" >> /etc/hosts'
```

---

## 実習ステップ

### STEP 1: 設定ファイルの作成

作業ディレクトリを作成し、必要なファイルを配置します。

1. **krb5.conf** (Kerberos 設定)

    ```ini
    [libdefaults]
        default_realm = TEST.LOCAL
        dns_lookup_realm = false
        dns_lookup_kdc = false

    [realms]
        TEST.LOCAL = {
            kdc = 127.0.0.1
            admin_server = 127.0.0.1
        }

    [domain_realm]
        .test.local = TEST.LOCAL
        test.local = TEST.LOCAL
    ```

2. **nginx.conf** (NGINX 設定)

    ```nginx
    load_module modules/ngx_http_auth_spnego_module.so;

    events {}

    http {
        server {
            listen 80;
            server_name nginx.test.local;

            location / {
                auth_gss on;
                auth_gss_realm TEST.LOCAL;
                auth_gss_keytab /etc/nginx/krb5.keytab;
                auth_gss_service_name HTTP/nginx.test.local;

                return 200 "SPNEGO Authentication Successful. User: $remote_user\n";
            }
        }
    }
    ```

### ✅ チェックポイント

- [ ] `krb5.conf` 内の `default_realm` が `TEST.LOCAL` になっていることを確認した
- [ ] `nginx.conf` で `ngx_http_auth_spnego_module.so` をロードしていることを確認した

### STEP 2: コンテナイメージのビルド

NGINX に SPNEGO モジュールを組み込むための Dockerfile を用意します。

1. **Dockerfile.kdc**

    ```dockerfile
    FROM ubuntu:24.04
    ENV DEBIAN_FRONTEND=noninteractive
    RUN apt-get update && apt-get install -y krb5-kdc krb5-admin-server
    COPY krb5.conf /etc/krb5.conf
    RUN kdb5_util create -s -P admin_password
    CMD ["krb5kdc", "-n"]
    ```

2. **Dockerfile.nginx**

    ```dockerfile
    FROM nginx:1.25.1 AS builder
    RUN apt-get update && apt-get install -y git build-essential libkrb5-dev wget libpcre3-dev zlib1g-dev
    RUN wget http://nginx.org/download/nginx-1.25.1.tar.gz && tar zxvf nginx-1.25.1.tar.gz
    RUN git clone https://github.com/stnoonan/spnego-http-auth-nginx-module.git
    WORKDIR /nginx-1.25.1
    RUN ./configure --with-compat --add-dynamic-module=../spnego-http-auth-nginx-module && make modules

    FROM nginx:1.25.1
    RUN apt-get update && apt-get install -y krb5-user && rm -rf /var/lib/apt/lists/*
    COPY --from=builder /nginx-1.25.1/objs/ngx_http_auth_spnego_module.so /etc/nginx/modules/
    ```

3. **docker-compose.yml**

    ```yaml
    version: "3"
    services:
      kdc:
        build:
          context: .
          dockerfile: Dockerfile.kdc
        ports:
          - "88:88"
          - "88:88/udp"
        hostname: kdc.test.local

      nginx:
        build:
          context: .
          dockerfile: Dockerfile.nginx
        ports:
          - "8080:80"
        volumes:
          - ./nginx.conf:/etc/nginx/nginx.conf:ro
          - ./krb5.keytab:/etc/nginx/krb5.keytab:ro
          - ./krb5.conf:/etc/krb5.conf:ro
        depends_on:
          - kdc
        hostname: nginx.test.local
    ```

### ✅ チェックポイント

- [ ] `docker-compose.yml` で `krb5.keytab` が NGINX にマウントされていることを確認した

### STEP 3: KDC の起動と Keytab の生成

Keytab ファイル（サーバー用パスワードファイル）がないと NGINX が起動できないため、まず KDC だけを起動して生成します。

```bash
# KDC 起動
podman compose up -d kdc

# ユーザープリンシパルの作成 (パスワード: userpass)
podman compose exec kdc kadmin.local -q "addprinc -pw userpass user1@TEST.LOCAL"

# NGINX 用の SPN 作成
podman compose exec kdc kadmin.local -q "addprinc -randkey HTTP/nginx.test.local@TEST.LOCAL"

# Keytab のエクスポートとホストへの取り出し
podman compose exec kdc kadmin.local -q "ktadd -k /tmp/krb5.keytab HTTP/nginx.test.local@TEST.LOCAL"
KDC_CID=$(podman ps -q --filter label=io.podman.compose.service=kdc | head -n1)
test -n "$KDC_CID" || { echo "kdc コンテナが見つかりません"; exit 1; }
podman cp "$KDC_CID":/tmp/krb5.keytab ./krb5.keytab
chmod 644 ./krb5.keytab
```

### ✅ チェックポイント

- [ ] カレントディレクトリに `krb5.keytab` が作成されたことを確認した

### STEP 4: NGINX の起動と動作確認

```bash
# NGINX 起動
podman compose up -d nginx

# 1. チケット(TGT)の取得
export KRB5_CONFIG=$(pwd)/krb5.conf
kinit user1@TEST.LOCAL
# パスワード: userpass

# 2. チケットの確認 (krbtgt/TEST.LOCAL@TEST.LOCAL があればOK)
klist

# 3. SPNEGO 認証によるアクセス
curl --negotiate -u : http://nginx.test.local:8080/
```

**成功時の出力:**
`SPNEGO Authentication Successful. User: user1@TEST.LOCAL`

### ✅ チェックポイント

- [ ] `klist` で有効なチケットが表示されていることを確認した
- [ ] `curl` の結果に自分のユーザー名が含まれていることを確認した

---

### STEP 5: Kerberos の相互認証 (Mutual Authentication) を理解する

`curl --negotiate` の背後では、クライアントだけでなくサーバー側も「正しいサービス鍵を持っていること」を示し、相互認証が成立します。

```mermaid
sequenceDiagram
    participant C as Client (user1)
    participant K as KDC (AS/TGS)
    participant S as Service (nginx)

    Note over C: K_user = userの長期鍵(パスワード由来)
    Note over K: K_tgs = krbtgt鍵, K_svc = HTTP/nginx.test.local鍵

    C->>K: AS-REQ (+ preauth: TS を K_user で暗号化)
    K-->>C: AS-REP (TGT{...}_K_tgs + K_c,tgs を K_user で暗号化)

    C->>K: TGS-REQ (TGT + Authenticator{...}_K_c,tgs + SPN)
    K-->>C: TGS-REP (Ticket_svc{...}_K_svc + K_c,s を K_c,tgs で暗号化)

    C->>S: AP-REQ (Ticket_svc + Authenticator{...}_K_c,s)
    S->>S: keytab から K_svc で Ticket_svc を復号
    S-->>C: AP-REP (TS+1 を K_c,s で暗号化)

    Note over C,S: C が AP-REP を K_c,s で復号できれば mutual auth 成立
```

#### どの鍵で何を暗号化・復号するか

1. **AS-REQ / AS-REP（TGT取得）**
   - クライアント: `K_user` で preauth（タイムスタンプ）を暗号化して送信。
   - KDC: 受信した preauth を `K_user` で復号して本人性を確認。
   - KDC: `K_c,tgs`（クライアント-TGS セッション鍵）を `K_user` で暗号化して返却。
   - KDC: TGT 本体は `K_tgs` で暗号化（クライアントは中身を読めない）。

2. **TGS-REQ / TGS-REP（サービスチケット取得）**
   - クライアント: Authenticator を `K_c,tgs` で暗号化。
   - KDC: TGT を `K_tgs` で復号し、`K_c,tgs` を取り出して Authenticator を復号。
   - KDC: `K_c,s`（クライアント-サービスセッション鍵）を `K_c,tgs` で暗号化して返却。
   - KDC: サービスチケット `Ticket_svc` を `K_svc` で暗号化（サービスだけが読める）。

3. **AP-REQ / AP-REP（実サービスアクセス + 相互認証）**
   - クライアント: `Ticket_svc` と Authenticator(`K_c,s` 暗号化) を送信。
   - サービス: keytab の `K_svc` で `Ticket_svc` を復号し、`K_c,s` を取得。
   - サービス: `K_c,s` で Authenticator を復号し、クライアント真正性を確認。
   - サービス: AP-REP（典型的には `TS+1`）を `K_c,s` で暗号化して返す。
   - クライアント: AP-REP を `K_c,s` で復号できれば、サーバー真正性を確認。

#### 手軽な確認方法 (`curl -v`)

`curl -v --negotiate` では、最終 `200 OK` 側にも `WWW-Authenticate: Negotiate <token>` が返ると、AP-REP が返却されている（= mutual 認証が成立している）ことを確認できます。

```bash
curl --negotiate -u : -v http://nginx.test.local:8080/ -o /dev/null
```

**参考出力（要点）**

```text
< HTTP/1.1 401 Unauthorized
< WWW-Authenticate: Negotiate
...
> Authorization: Negotiate YIIF...   # クライアントの AP-REQ
< HTTP/1.1 200 OK
< WWW-Authenticate: Negotiate YIGC... # サーバーの AP-REP（mutual 認証）
```

`200 OK` なのに後者の `WWW-Authenticate: Negotiate <token>` が無い場合は、相互認証トークンが返っていない可能性があります（モジュール設定・実装差分を確認）。

---

## 片付け

```bash
podman compose down
rm krb5.keytab
# /etc/hosts の編集内容は必要に応じて手動で削除してください
```

---

## 参考文献

- [MIT Kerberos Documentation](https://web.mit.edu/kerberos/krb5-latest/doc/)
- [spnego-http-auth-nginx-module](https://github.com/stnoonan/spnego-http-auth-nginx-module)
- [RFC 4559 - SPNEGO-based Kerberos HTTP Authentication in Microsoft Windows](https://datatracker.ietf.org/doc/html/rfc4559)

---

## 🔧 トラブルシューティング

### NGINX が起動しない / エラーログが出る

- **原因**: Keytab ファイルの読み込み権限がない、またはパスが間違っている。
- **対処**: `chmod 644 ./krb5.keytab` を実行し、`nginx.conf` の `auth_gss_keytab` パスを再確認してください。

### curl で 401 Unauthorized になる

- **原因**: ホストマシンの `kinit` でチケットを取得していない、または `KRB5_CONFIG` が正しく設定されていない。
- **対処**: `klist` でチケットの有無を確認し、`export KRB5_CONFIG=$(pwd)/krb5.conf` を実行した同一ターミナルで `curl` を実行してください。

### `no container with name or ID "kadmin.local" found` が出る

- **原因**: `podman exec -it $(podman ps -q -f name=kdc) ...` の `$(...)` が空になり、`kadmin.local` がコンテナ名として解釈される。
- **対処**: `podman compose exec kdc ...` を使う。`podman cp` が必要な場合のみ `podman ps -q --filter label=io.podman.compose.service=kdc` でコンテナ ID を取得する。

---

## 💻 環境別注意事項

### WSL2 の場合

- ホスト Windows 側のブラウザからアクセスする場合、Windows 側でも `hosts` 設定と Kerberos 設定が必要になります。実習は WSL2 のターミナル内 (`curl`) で完結させることを推奨します。
