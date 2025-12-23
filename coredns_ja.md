# CoreDNS 実習：親子 DNS サーバーを構築して名前解決を理解する

ソフトウェアエンジニア向けに、DNS の基本的な仕組み（権威サーバー、フォワーディング、委譲）を、CoreDNS を使って実際に構築しながら学びます。

## ゴール

以下の構成を **手を動かして構築し、理解** します。

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

**学ぶこと:**

1. **権威 DNS サーバー (Authoritative Server):** 自分のドメイン（ゾーン）の情報を持ち、問い合わせに答えるサーバー。
2. **フォワーディング (Forwarding):** 自分が知らないドメインの問い合わせを、別のサーバーに転送する仕組み。
3. **ゾーン階層:** 親 (`sokoide.com`) と子 (`foo.sokoide.com`) の関係性。

※ 本実習は分かりやすさのため **フォワーディング** を使います。実際のDNS階層は **委譲（NS + グルーレコード）** が基本で、最後の「次のステップ」で扱います。

---

## 前提条件

- **VM 2台** (Ubuntu 24.04 推奨)
  - **VM1 (Parent):** IP `192.168.100.10`
  - **VM2 (Child):** IP `192.168.100.20`
  - ※ IPアドレスが異なる場合は、以降の手順の IP を適宜読み替えてください。
- **ツール:** `curl`, `tar`, `dig` (dnsutils)

**事前準備 (両方の VM で実行):**

```bash
sudo apt update && sudo apt install -y curl tar dnsutils
```

---

## Step 1. CoreDNS のインストール

CoreDNS は Go 言語で書かれた単一バイナリの DNS サーバーです。依存関係がなく、導入が非常に簡単です。

**VM1, VM2 両方で実行:**

```bash
# CoreDNS のダウンロード
CORE_VERSION="1.13.2"
curl -L "https://github.com/coredns/coredns/releases/download/v${CORE_VERSION}/coredns_${CORE_VERSION}_linux_amd64.tgz" -o coredns.tgz

# 展開と配置
tar -xzvf coredns.tgz
sudo mv coredns /usr/local/bin/

# 動作確認
coredns -version
```

---

## Step 2. VM1 (親) の構築: sokoide.com

VM1 は親ドメイン `sokoide.com` を管理します。また、子ドメイン `foo.sokoide.com` への問い合わせが来た場合、VM2 へ転送するように設定します。

**VM1 (`192.168.100.10`) で実行:**

1. 作業ディレクトリ作成

   ```bash
   mkdir -p ~/coredns_parent && cd ~/coredns_parent
   ```

2. **Corefile** (設定ファイル) 作成

   ```bash
   cat <<'EOF' > Corefile
   sokoide.com:10053 {
       file db.sokoide.com
       log
       errors
   }

   foo.sokoide.com:10053 {
       # 問い合わせを VM2 (Child) へ転送
       forward . 192.168.100.20:10053
       log
       errors
   }
   EOF
   ```

3. **ゾーンファイル** (レコード定義) 作成

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

## Step 3. VM2 (子) の構築: foo.sokoide.com

VM2 はサブドメイン `foo.sokoide.com` を管理します。ここには具体的なレコード（例: `test`）を登録します。

**VM2 (`192.168.100.20`) で実行:**

1. 作業ディレクトリ作成

   ```bash
   mkdir -p ~/coredns_child && cd ~/coredns_child
   ```

2. **Corefile** 作成

   ```bash
   cat <<'EOF' > Corefile
   foo.sokoide.com:10053 {
       file db.foo.sokoide.com
       log
       errors
   }
   EOF
   ```

3. **ゾーンファイル** 作成

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

## Step 4. DNS サーバーの起動

それぞれの VM で CoreDNS を起動します。ログを確認するため、このターミナルは開いたままにしてください（または `screen`/`tmux` やバックグラウンド実行 `&` を利用）。

**VM1 (Parent) ターミナル:**

`/usr/local/bin` が PATH に入っていて、特権ポートではない `10053` を使う場合は `sudo` なしでも起動できます。

```bash
cd ~/coredns_parent
sudo /usr/local/bin/coredns -conf Corefile
```

**VM2 (Child) ターミナル:**

```bash
cd ~/coredns_child
sudo /usr/local/bin/coredns -conf Corefile
```

---

## Step 5. 動作確認 (dig)

別のターミナル（またはローカル PC）から `dig` コマンドを使って検証します。

### 1. 親ゾーンの正引き

親サーバー (VM1) に `www.sokoide.com` を問い合わせます。

```bash
dig @192.168.100.10 -p 10053 www.sokoide.com +short
```

> **結果:** `1.1.1.1` が返れば成功。

### 2. 子ゾーンの解決 (Forwarding)

親サーバー (VM1) に、子ゾーンにある `test.foo.sokoide.com` を問い合わせます。

```bash
dig @192.168.100.10 -p 10053 test.foo.sokoide.com
```

**出力の読み方:**

```text
;; ANSWER SECTION:
test.foo.sokoide.com.   3600    IN      A       2.2.2.2  <-- 正しいIP (VM2から取得)

;; SERVER: 192.168.100.10#10053(192.168.100.10)          <-- 親サーバーが答えている
```

### 何が起きたか？

1. クライアントは VM1 (`sokoide.com`) に問い合わせた。
2. VM1 は設定 (`forward . 192.168.100.20`) に従い、VM2 に問い合わせを転送した。
3. VM2 が `2.2.2.2` と回答した。
4. VM1 がその結果をクライアントに返した。

VM1 のログを見ると、転送が行われた様子（ログ出力設定によりますが）やアクセスが確認できます。

---

## クリーンアップ

実習終了後の後片付けです。

1. **プロセス停止:** 各 VM で起動している `coredns` を `Ctrl+C` で停止します。
2. **ファイル削除:**

   ```bash
   # 両方の VM で
   rm -rf ~/coredns_parent ~/coredns_child
   # 必要ならバイナリも削除
   sudo rm /usr/local/bin/coredns
   ```

---

## 次のステップ

今回は **フォワーディング (Forwarding)** を使いましたが、インターネット上の DNS の基本は **委譲 (Delegation)** です。
委譲を行う場合は、親のゾーンファイル (`db.sokoide.com`) に、子の NS レコード (`foo IN NS ns1.foo...`) とグルーレコード (`ns1.foo IN A ...`) を書くことで、クライアント自身に次のサーバーへ問い合わせに行かせることができます。

CoreDNSの設定を変えて、委譲の挙動を試してみるのも良い学習になります。
