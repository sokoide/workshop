# ワークショップ改善計画 (Improvement Plan)

対象: ジュニア SWE(経験 1〜3 年)向け学習素材としての `infra/`・`software/`
評価軸: **Easy to follow(追従容易) / Simple(単純) / Concise(簡潔) / Correct(正確)**
作成日: 2026-10-07(全文レビュー: infra 27 本・software 16 本・assets 19 ディレクトリ)

> この文書は計画である。Phase 0（重大修正）は実施済み、Phase 1（テンプレート）は `docs/TEMPLATE.md` として作成済み、Phase 2（全ワークショップへのテンプレート適用、43 ペア）も実施済み（コミット c01cd31）。Phase 3（正確性検証パス: コマンド実機検証・asset コードバグ修正 3 件・ドキュメント約 30 件修正、コミット 015b6dc）、Phase 4（簡潔性パス: tcpip_stack・web_security 分割、重複削除、カタログ化、コミット 2fb96b0）、Phase 5（学習パス整備: WS1/WS2 実行可能化・README 難易度別再構成、コミット bccd480）、Phase 6（アセット品質: README 20 件追加・テスト追加で ws1/ws2 infra カバレッジ 94.7%/93.1%・podman/モジュール名統一・oauth2 compose 同梱、コミット fc8db8c）、Phase 7（自動検証強化: check_links/check_glossary_anchors/check_asset_paths/check_workshop_skeleton + `make check-docs`/`make check` 集約・骨格完全化 38 ファイル、コミット 5662d01）まで全フェーズ実施済み。

---

## 1. 現状評価サマリ

### 1.1 強み(このまま維持するもの)

- 日英ペア完全(54 ファイル)。`make check-headings` による構造検証、`make check-go` による全モジュール build/vet/race 検証など保守基盤が優秀
- ゴールドスタンダードとなる良質ドキュメントが存在:
  - `infra/cache_en.md`(224 行・期待出力・cleanup・参考文献付き)
  - `software/idempotency_en.md`(所要時間・用語リンク・チェックポイント・トラブルシュート完備)
  - `infra/kerberos_en.md`・`coredns_en.md`・`tls_en.md`・`envoy_en.md`・`vlan_en.md`・`rabbitmq_en.md`・`secret_management_en.md`
- 良質アセット: `idempotency`(二重チェック+`context.WithoutCancel`+`errors.Join`)、`rate_limiting`(Lua+miniredis テスト)、`tcpip_stack`、`bbs`(context 伝播トランザクション)
- コミット済み機密情報なし(`.gitignore` が `*.crt/*.key/*.pem/.env` をカバー)

### 1.2 横断的なギャップ(検証済み数値)

| 骨格要素                  | infra(27本) | software(15本) | 備考                                                   |
| ------------------------- | ----------- | -------------- | ------------------------------------------------------ |
| Goal セクション           | —           | —              | 計42本中31本のみ                                       |
| Prerequisites             | 8/27        | ほぼ0          | 書式も `### 1.`/`##`/`### ✅` と不統一                 |
| Cleanup                   | 10/27       | —              | supply_chain・oauth2_entra(有償!)・k8s_operations なし |
| References                | 18/27       | —              | —                                                      |
| Summary/What you learned  | —           | —              | 計42本中10本のみ                                       |
| 期待出力(Expected output) | 4/42        | —              | ジュニアの自己検証が不可能                             |
| 所要時間                  | 3/42        | —              | idempotency のみ45分表記                               |
| チェックポイント          | 20/27       | 1/15           | セクション間で文化が違う                               |

その他:

- **学習曲線の逆転**: WS8(最小・実行可能・最良の入門素材)がシリーズ最後。WS1/WS2 は `your-project` プレースホルダと擬似 `LDAPClient` で実行不能
- **難易度表示なし**・推奨学習順なし(README はフラットな箇条書き)
- **Go バージョン乱立**: 21 モジュールが 1.21〜1.26.2(CLAUDE.md は 1.26.4 と記載)
- **長大ドキュメント**: tcpip_stack 1,743 行 / web_security 1,460 行(うち約 870 行は victim アプリに適用しないままの対策解説)/ kubernetes_basics 956 行
- **アセット**: README は bbs のみ(1/19)。テストゼロが 9/19(CLAUDE.md の >80% カバレッジ目標と乖離)
- **podman/docker 不統一**: supply_chain は全編 docker(リポジトリ規約は Podman)。traefik の Makefile は `docker compose`、ドキュメントは `podman compose`。traefik WS は v2.10、tls WS は v3.1

---

## 2. 欠陥リスト(フェーズ0対応の重大項目)

### 2.1 ビルド破損・実行不能(最優先)

| #   | 場所                                        | 欠陥                                                                                                                                                                   | 修正方針                                                                                                          |
| --- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| B1  | `.gitignore:11` `pb/`                       | `infra/assets/http_persistent_conn/pb/greeter_grpc.pb.go` が無視され未コミット。fresh clone で `go run main.go` が失敗(`pb.RegisterGreeterServer` 未定義)              | `pb/` 行を削除し `http_persistent_conn/pb/` 配下を明示コミット。`git status --ignored` で他要因がないか全资产確認 |
| B2  | `software/clean_arch_ws7_en.md:205,207-209` | `go install github.com/matyer/jwt-cli@latest` と `go run github.com/golang-jwt/jwt/v5/cmd/jwt@latest` の**両方が実在せず**検証ステップが停止                           | 実在する手段(jwt.io 表示例、`docker run` による CLI、またはリポジトリ内 `cmd/jwtgen` ミニツール)に置換            |
| B3  | `software/secure_coding_en.md:672-674`      | `WithdrawDB` 内の `ctx` が未定義でコンパイル不可+`context.Context` 第1引数規約違反                                                                                     | シグネチャを `func WithdrawDB(ctx context.Context, ...)` に修正                                                   |
| B4  | `software/clean_arch_ws4_en.md:172`         | `entity.NewThread(board.ID, in.Title)` は2引数だが実装は3引数 `NewThread(boardID, title, owner)`(`assets/bbs/.../thread.go:21`)                                        | スニペットを実装に合わせ修正。同ドキュメント273行目の switch を含まない裸 `case` も修正                           |
| B5  | `infra/tcpip_stack_en.md`                   | `#cgo LDFLAGS: -lpcap` が libpcap 未使用なのにリンク(libpcap-dev 未指定で build 失敗)。STEP 5 の main.go が STEP 6 で作る `pkg/udp` を import し手順順でコンパイル不能 | LDFLAGS 削除。STEP 順を入れ替え/依存関係を明記                                                                    |
| B6  | `infra/rate_limiting_en.md` STEP 4          | サーバー未起動のまま `curl` を実行させるため必ず失敗                                                                                                                   | サーバ起動コマンドを先行に挿入                                                                                    |
| B7  | `infra/k8s_lb_en.md`                        | iptables ルールの挿入順序により STEP 6 が失敗。検証 `curl` の本体欠落                                                                                                  | ルール順を検証して再構成                                                                                          |
| B8  | `infra/iscsi_pv_en.md`                      | `test-pod.yaml` の中身が一度も示されない                                                                                                                               | マニフェスト全文を掲載                                                                                            |
| B9  | `infra/zero_trust_en.md`                    | 検証 Pod が一度もデプロイされないまま検証手順が進む                                                                                                                    | デプロイ手順を補完                                                                                                |
| B10 | `infra/kubernetes_basics_en.md`             | クラスタ名不整合・未作成マニフェストを apply・`COPY go.mod go.sum` だが go.sum 無し                                                                                    | 全手順を kind で実行検証しながら修正                                                                              |

### 2.2 教育内容の誤り(概念の汚染)

| #   | 場所                                                                                  | 欠陥                                                                                                                                                                                                                                                                                                                                  | 修正方針                                                                                                        |
| --- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| C1  | `software/clean_arch_ws3_en.md:85,97-104,245`                                         | リポジトリポートを「Domain Ports(`internal/domain/port/`)」と教えるが、実体は `assets/bbs/internal/usecase/repository.go`(UseCase 層)。`clean_arch_en.md:333` と正反対                                                                                                                                                                | 「UseCase-owned ports」に統一。WS3〜WS7 全てでポート所有権の記述を棚卸し                                        |
| C2  | `software/solid_en.md:79-81`                                                          | `Notifier` を domain port とするが、`clean_arch_en.md:331` は通知を UseCase port と明言                                                                                                                                                                                                                                               | clean_arch.md の単一真実源に合わせて修正                                                                        |
| C3  | `software/clean_arch_ws8_en.md:24-29`                                                 | ディレクトリツリーが実 asset(greeting、`internal/` 無しフラット構成)と不一致                                                                                                                                                                                                                                                          | 実物に合わせる                                                                                                  |
| C4  | `software/clean_arch_ws6_en.md:56-58`                                                 | `port.ThreadRepository` 接頭辞が実 asset(パッケージ `usecase`)・WS4 と不一致。WS6 版 `Execute` が WS4 で追加した `CanPost` を黙って落とす                                                                                                                                                                                             | asset と突合して統一。「WS4 の成果を引き継ぐ」前提を明記                                                        |
| C5  | `software/clean_arch_ws2_en.md:96`                                                    | `getFromCache(id)`/`saveToCache(user)` が未定義のまま                                                                                                                                                                                                                                                                                 | 定義を提示 or asset 参照                                                                                        |
| C6  | `infra/assets/redis_leaderboard`                                                      | `AddScore` が `ZAdd`(スコア置換)で `ZIncrBy`(加算)ではない。命名と動作が不一致。`GetRank` が BAN 利用者を考慮しない                                                                                                                                                                                                                   | `ZIncrBy` に変更(または命名変更)+ドキュメントの説明を修正                                                       |
| C7  | `infra/assets/secret_management/.../vault/client.go`                                  | `WaitForVault` が sealed(HTTP 503)も ready 判定。レスポンス body 未 close                                                                                                                                                                                                                                                             | ステータスコード検証を追加                                                                                      |
| C8  | `software/assets/go_mq_patterns/pattern5_retry_dlq/main.go`                           | `failCount++` のデータ競合(2ワーカーから無同期)+ `Stop()` close 済み channel への send で panic の可能性                                                                                                                                                                                                                              | `atomic`/mutex 導入。`-race` で検証(`make check-go` が既に持つ検査で捕獲できる状態に)                           |
| C9  | `software/clean_arch_en.md:281` ほか                                                  | 自前規約「error を無視しない」違反が複数(`json.NewEncoder.Encode` の error 無視、`secure_coding:388` の `_, _ = rand.Read(b)`)                                                                                                                                                                                                        | 全ドキュメントのコード断片を規約準拠に                                                                          |
| C10 | `infra/glossary_en.md`                                                                | `<a name=...>` アンカーは network/container/kubernetes/security/mesh の5つのみ。`#cache`(cache)・`#mq`(rabbitmq)・`#storage`(minio)・`#architecture`(traefik/terraform)・`#oauth2` 系・`#protocol`(tcpip_stack:5) は**全て破綻**。109-117行・236-240行に誤配置、Exchange/Clean Architecture の定義重複。TLS/CA/Vault は項目自体が無い | アンカーを全セクションに付与。誤配置・重複を解消し、欠落用語を追加。**アンカー検証を Phase 7 の CI に組み込む** |
| C11 | `infra/oauth2_en.md:329` + `_ja.md` 同位置                                            | `/Users/scott/repo/sokoide/workshop/...` 個人絶対パスの markdown リンク                                                                                                                                                                                                                                                               | 相対パスに修正                                                                                                  |
| C12 | `infra/assets/minio/main.go:42` / `go_cache_patterns` + `go_cache_patterns_en.md:422` | deprecated な `EndpointResolverWithOptions` / 廃止済み `go-redis/redis/v8`(他は v9)                                                                                                                                                                                                                                                   | `BaseEndpoint` / `redis/go-redis/v9` に統一(ドキュメントも)                                                     |

### 2.3 構造・簡潔性の問題

- WS3〜WS7 が依存図・エラー境界・「Comparison: Without Layer Separation」・Prerequisites を毎回ほぼ同一内容で反復。WS3 は前半 265 行が `clean_arch.md` と BBS README の重複 → 共有イントロに切り出せば software 全体の 2〜3 割減
- `go_cache_patterns_en.md`/`go_mq_patterns_en.md` はタイトルに "In Go" とありながら**Goコードが1行も無い**(mermaid と asset パス参照のみ)。冒頭 190 行の MQ 一般論も両者で重複。「ワークショップ」ではなく「カタログ」として再定義するか、要点(Go コード)をインライン化
- `infra/web_security_en.md` 496-1370 行(約 870 行)の対策解説が victim アプリに適用されないまま。「適用して再検証」のハンズオンに変換 or 別ドキュメント(リファレンス)に分離
- `infra/oauth2_en.md` に「introspection 不使用」断りが 5 回以上重複
- `infra/http_persistent_en.md` のソケット監視スニペット 6 回重複、`podman-compose` と `podman compose` が混在
- `sharding_en.md` は Sparse Table/Segment Tree(アルゴリズム)と ES/Cortex(分散システム)という別トピックが混在。2 分割を検討
- CLAUDE.md のディレクトリツリーが `software/assets/` 層を省略(`software/advent-of-calm-2025` と記載)
- 日英ドリフト: ws3 は英語版 17 コードブロック vs 日本語版 14。**全欠陥は日英両方に存在**(修正は必ず両方へ)

---

## 3. フェーズ別実行計画

### Phase 0: 緊急修正(見積り: 1〜2日)

**目標: fresh clone で全ワークショップがビルド可能・主要検証ステップが実行可能**

1. B1〜B12 のビルド破損・実行不能を修正(日英両方)
2. C1〜C5 の概念誤り(ポート所有権・asset 不一致)を修正 — WS3 の修正が最優先(シリーズ全体の概念汚染源)
3. C11 の個人情報(絶対パス)除去
4. 各修正後に `make format`・`make check-headings`・`make check-go` を実行

**完了基準**: fresh clone で `make check-go` が全モジュール pass。http_persistent_conn が `go run` 可能。

### Phase 1: ワークショップテンプレート定義(見積り: 半日)

`idempotency_en.md` と `cache_en.md` を原型とした標準骨格を定義し、`docs/TEMPLATE.md` として明文化:

```markdown
# <トピック> Workshop: <一言サマリ>

> ⏱️ Estimated Time: XX min | Level: Beginner/Intermediate | OS: Linux/macOS
> 💡 Glossary: [用語1](../infra/glossary_en.md#<anchor>) ...

## Goal(達成状態を図示)

## Prerequisites(ツール+バージョン+環境+費用有無)

## Steps(STEP N + 各STEPに ✅ Checkpoint + 期待出力)

## What you learned

## Troubleshooting

## Cleanup

## References
```

- Level(初級/中級)・所要時間・必要 OS・费用の 4 項目を必須メタデータに
- `check_heading_structure.py` を拡張し、骨格セクションの有無を lint 化(既存ツールの延長で低コスト)

### Phase 2: 構造パス(全42ドキュメントへ適用)(見積り: 5〜7日)

優先順位(効果の高い順):

1. **Prerequisites 全ファイルへの追加**(8/27→27/27): 特に cache・web_security・oauth2_entra(Azure 費用に言及なし)・iscsi(kubectl 未記載)・zero_trust(kubectl/istioctl バージョン)
2. **Cleanup 全ファイルへの追加**(10/27→27/27): 最優先は oauth2_entra(有償クラウドの App 登録放置)・kubernetes_operations・supply_chain
3. **期待出力の追加**: 主要コマンド全てに `# Expected output:` コメント(cache_en.md 流)
4. **所要時間・Level の記載**(3/42→42/42)
5. **What you learned セクション**(10/42→42/42)
6. **チェックポイントの software 側への導入**(1/15→15/15。idempotency 流)

作業はドキュメント単位で分業可能。1 ファイル完了ごとに日英ペアで同時適用。

### Phase 3: 正確性検証パス(見積り: 5日)

- Phase 0 で対応しなかった残りドキュメントのコマンドを**全て実際に実行**して検証(特に kubernetes_basics・kubernetes_operations・http_persistent・load_balancer・container_runtime)
- ドキュメント記載のディレクトリ構造・行番号参照・ファイル名と asset 実物の突合(minio・rate_limiting・web_security に既知ドリフト)
- `make check-go GO_CHECK_FLAGS=--require-integration` を CI 相当で実行
- 検証結果を `git notes` に記録(リポジトリ運用に従う)

### Phase 4: 簡潔性パス(見積り: 3〜5日)

1. **長大ドキュメントの分割**:
   - tcpip_stack(1,743 行)→ パート 1: Ethernet/IP/ICMP、パート 2: UDP(kubernetes_basics の「30-45 分/セクション」警告がモデル)
   - web_security(1,460 行)→ 攻撃体験編 + 防御実践編(870 行の理論を適用演習に変換)
2. **重複の削除**: WS3〜WS7 の共有イントロを `clean_arch.md` へ集約。oauth2 の断り重複・http_persistent のスニペット重複を圧縮
3. **カタログの明確化**: go_cache/go_mq に「リファレンス型」の位置づけを明記し、各パターンの核心コード(例: singleflight の `DoChan` 呼び出し部)だけをインライン表示
4. **sharding の分割**検討(アルゴリズム側/分散システム側)
5. glossary の誤配置・重複解消(C10 と連動)

### Phase 5: 学習パス整備(見積り: 1〜2日)

1. **WS8 を WS1 の前に**移動(または「はじめに読むべき WS0」として明示)。README の並びも変更
2. README への**難易度・推奨順**の導入。例:

   ```text
   初級(最初の1週間): clean_arch.md → WS8 → WS1 → idempotency → cache → rate_limiting → tls → traefik
   中級(2〜4週目): WS2-WS7 → rabbitmq → secret_management → minio → envoy → load_balancer
   上級: tcpip_stack → k8s_lb → zero_trust → supply_chain → kubernetes_operations
   ```

3. 依存関係の明示(例: kubernetes_operations は kubernetes_basics の後、envoy は traefik の後)
4. WS1/WS2 を実行可能な形にリライト(開始コードを `assets/` に用意し、`your-project` プレースホルダを廃止)

### Phase 6: アセット品質(見積り: 5日)

1. **全 asset に README.md を追加**(実行手順・必要ツール・ドキュメント本文への逆リンク。bbs の README_ja.md が原型)
2. **oauth2 asset に Keycloak compose を同梱**(現在はドキュメントの heredoc で生成)
3. **テスト追加**: 9 つのテスト無し asset のうち、教育価値の高い go_mq_patterns(データ競合修正とセット)・minio・oauth2 から。全 asset でなくても、CLAUDE.md の目標を「ドメイン/usecase 層はテスト必須、main.go の配線は任意」に現実化
4. **Go バージョン方針の決定**: 21 モジュール(1.21〜1.26.2)を「教育上の最低バージョンはドキュメント明記・モジュールは現状維持(AGENTS.md の『auto-update 禁止』を尊重)」とし、CLAUDE.md の記載を実態(1.21+)に修正
5. **podman/docker 統一**: supply_chain を podman 表記に、traefik Makefile を `podman compose` に。traefik WS のバージョンを v3.x に統一(tls WS と揃う)
6. **モジュール名の統一**(`github.com/sokoide/workshop/...` へ)
7. web_security asset に「意図的脆弱性」の警告 README を追加

### Phase 7: 自動検証の強化(見積り: 2〜3日)

既存 `make` ターゲットの延長で:

1. **glossary アンカー検証**: ドキュメント内 `glossary_en.md#xxx` リンクと glossary 側アンカーの突合スクリプト(check_heading_structure.py と同じディレクトリに配置)
2. **ドキュメント↔asset 突合**: ドキュメント掲載パス(`assets/...`)の実在検証+主要スニペットの存在検証(完全同期は過剰。ファイルパスとシグネチャ程度)
3. **相対リンク検証**(C11 の再発防止。`/Users/` 等の絶対パス検出を含む)
4. **構造 lint**(Phase 1 の骨格チェック)
5. 上記を `make check-docs` に集約し、CI 化

---

## 4. ファイル別アクション表(優先度順)

### P0(Phase 0 で対応)

| ファイル/対象                                          | 主なアクション                            |
| ------------------------------------------------------ | ----------------------------------------- |
| `.gitignore` + `infra/assets/http_persistent_conn/pb/` | B1 ビルド修正                             |
| `software/clean_arch_ws3_{en,ja}.md`                   | C1 ポート所有権の誤り修正(概念汚染の根源) |
| `software/clean_arch_ws7_{en,ja}.md`                   | B2 JWT コマンド差し替え                   |
| `software/secure_coding_{en,ja}.md`                    | B3 ctx 修正 + C9 rand.Read                |
| `software/clean_arch_ws4_{en,ja}.md`                   | B4 NewThread + 裸 case                    |
| `infra/tcpip_stack_{en,ja}.md` + asset                 | B5 LDFLAGS + STEP 順                      |
| `infra/glossary_{en,ja}.md`                            | C10 アンカー全面修正                      |
| `infra/oauth2_{en,ja}.md`                              | C11 絶対パス                              |
| `software/assets/go_mq_patterns/pattern5_retry_dlq`    | C8 データ競合                             |
| `infra/assets/redis_leaderboard`                       | C6 AddScore                               |
| `infra/assets/secret_management`                       | C7 WaitForVault                           |
| `software/solid_{en,ja}.md`                            | C2 Notifier                               |
| `software/clean_arch_ws8_{en,ja}.md`                   | C3 ツリー                                 |
| `software/clean_arch_ws6_{en,ja}.md`                   | C4 接頭辞+CanPost                         |
| `software/clean_arch_ws2_{en,ja}.md`                   | C5 未定義ヘルパー                         |
| `infra/kubernetes_basics_{en,ja}.md`                   | B10 全手順実行検証                        |

### P1(Phase 2〜3)

`rate_limiting`・`k8s_lb`・`iscsi_pv`・`zero_trust`(検証手順修正)、`minio`+asset・`web_security`+asset(ドリフト解消)、`oauth2_entra`(費用+cleanup)、`supply_chain`(podman 統一+cleanup)、`kubernetes_operations`(cleanup)、`traefik`(バージョン統一+STEP 4 完結)、`clean_arch_en`(Encode)、`clean_arch_ws5`(err 分岐)

### P2(Phase 4〜6)

`web_security`(分割)・`tcpip_stack`(分割)・`go_cache_patterns`/`go_mq_patterns`(カタログ化)・`sharding`(分割検討)・WS3〜WS7(重複削除)・全 asset(README・テスト)

---

## 5. 完了基準(Acceptance Criteria)

1. fresh clone で `make check-go` が全モジュール pass、http_persistent_conn が実行可能
2. 全 42 ドキュメントに骨格 7 要素(Goal/Prerequisites/Steps+Checkpoint/期待出力/What you learned/Cleanup/References)とメタデータ(所要時間・Level・OS)がある
3. 全ワークショップの検証コマンドが記載どおり実行して成功する(期待出力付き)
4. glossary へのリンクが全て有効(自動検証 pass)
5. 全 asset に README があり、asset ディレクトリ単独で実行手順が分かる
6. WS8 が学習順の先頭に配置され、README に難易度別の推奨パスがある
7. ドキュメント内コードがリポジトリ規約(error 無視禁止・context 第 1 引数)に準拠
8. 日英ペアの構造差異ゼロ(`make check-headings` + コードブロック数一致)

## 6. ガードレール(やってはいけないこと)

- **go.mod の無闇なバージョンアップ禁止**(AGENTS.md の規約)。まず CLAUDE.md 側を実態に合わせる
- 日英ペアの片方だけの編集禁止(必ず同時コミット)
- 良質ドキュメント(cache・idempotency・kerberos・coredns・tls・envoy・vlan)の大幅改変は不要。テンプレートの原型として温存
- web_security・supply_chain の「意図的脆弱性」は教育上必要。修正ではなく警告表示で対応
- 一度に全フェーズを適用しない。Phase 0→1→2 の順に、各フェーズで `make format`/`check-headings`/`check-go` を通してからコミット

## 7. タイムライン概算

| フェーズ             | 見積り | 依存            |
| -------------------- | ------ | --------------- |
| Phase 0 緊急修正     | 1〜2日 | —               |
| Phase 1 テンプレート | 半日   | —               |
| Phase 2 構造パス     | 5〜7日 | Phase 1         |
| Phase 3 正確性検証   | 5日    | Phase 0(並行可) |
| Phase 4 簡潔性       | 3〜5日 | Phase 2         |
| Phase 5 学習パス     | 1〜2日 | Phase 2         |
| Phase 6 アセット     | 5日    | 並行可          |
| Phase 7 自動検証     | 2〜3日 | Phase 1         |

合計: 約 4〜5 週間(1 人の場合)。Phase 0+1 だけで大きな改善効果が見込める。
