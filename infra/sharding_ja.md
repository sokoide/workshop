# シャーディング: 原理から Go 実装まで

このドキュメントは、シャーディングを「概念説明」ではなく「実装して運用できる設計」として整理します。

扱う内容:

1. シャーディングが必要になる理由
2. 戦略選定（hash / range / tenant）
3. fan-out・ホットスポット・再シャーディング
4. 制御プレーン安定性の重要性
5. Sparse Table / Segment Tree とシャーディングの関係
6. Go での実装パターン
7. Elasticsearch / Cortex への対応づけ

---

## 1) シャーディングとは何か

シャーディングは、データとトラフィックを複数ノードへ水平分割する設計です。

単一ノード集中:

```text
all data -> one machine
```

分割後:

```text
data partitioned -> many machines
```

各分割単位を **shard** と呼びます。

必要になる背景:

- 保存容量の上限
- 書き込みスループットの上限
- クエリ同時実行数の上限
- 障害時の blast radius 拡大

ただし、スケールと引き換えに次の課題が増えます。

- クロスシャード fan-out
- スキューとホットスポット
- オンライン再分配の複雑化
- ルーティング整合性
- 制御プレーン収束の不安定化

---

## 2) 最初に固定すべきコア概念

### 2.1 シャードキー

「このデータはどのシャードが持つか」を決める軸です。

悪いキー選定は:

- 負荷偏り
- ホットパーティション
- 高コストな多シャード検索

につながります。

### 2.2 ルーティング関数

キーが決まると、ルーティングは次です。

```text
route(key) -> shard
```

代表パターン:

- modulo hash
- consistent hash ring
- range map
- tenant/domain map

### 2.3 レプリケーション

シャーディングは分散、レプリケーションは可用性です。

組み合わせにより以下が決まります。

- 書き込みクォーラムと耐久性
- 読み取り整合性
- 部分障害時の挙動

---

## 3) 戦略選定: Hash / Range / Tenant

### 3.1 ハッシュ分割

キーをハッシュして振り分けます。

向いているケース:

- 高書き込み
- キー分布が比較的均一

トレードオフ:

- 書き込み分散はしやすい
- 集計クエリは fan-out しやすい

### 3.2 レンジ分割

時刻帯や ID 範囲で分割します。

向いているケース:

- 時系列ウィンドウ検索
- 範囲スキャン

トレードオフ:

- 範囲読み取りが効率的
- 最新レンジがホット化しやすい

定番対策:

- time partition + hash suffix

### 3.3 テナント分割

tenant/org/customer 境界で分割します。

向いているケース:

- マルチテナント SaaS
- ノイジーネイバー分離
- テナント別 SLO

トレードオフ:

- 大規模テナントが容量を圧迫しやすい

定番対策:

- shuffle sharding
- テナント内ハッシュ

---

## 4) 一貫性ハッシュと再配置コスト

単純な modulo 方式:

```text
hash(key) % N
```

問題:

`N` の変更で多くのキーが再配置される。

一貫性ハッシュはスケール時の移動量を抑えます。

実装パターン:

- 仮想ノード付きトークンリング
- Rendezvous (HRW) hashing
- Jump Consistent Hash

運用上の要点:

- 仮想ノード数で分布平滑化
- レプリカ配置はトポロジを考慮
- スケールイベントは段階実行

---

## 5) Sparse Table / Segment Tree とシャーディングの関係

これらはシャーディング方式そのものではありません。**シャード制御判断を高速化する制御プレーン向けデータ構造**です。

### 5.1 どこで使うか

- データプレーン:
  - 実トラフィックの write/read
- 制御プレーン:
  - シャード状態スナップショット
  - 負荷ベース選択
  - 再配置判断

Sparse Table / Segment Tree は制御プレーンで効きます。

### 5.2 Sparse Table（静的レンジクエリ向け）

シャードメタデータが「一定期間ほぼ不変」の場合に使います。

性質:

- 構築: `O(n log n)`
- クエリ（min/max など）: `O(1)`
- 更新: 重い（再構築前提）

シャーディング文脈での用途:

- 期間スナップショットから最小遅延シャードを即時参照
- バッチ更新されるプランナーメタデータの高速参照
- 静的区間統計に基づく候補シャード抽出

### 5.3 Segment Tree（動的レンジクエリ向け）

シャード負荷メトリクスが継続的に変化する場合に使います。

性質:

- 構築: `O(n)` もしくは `O(n log n)`
- 一点更新: `O(log n)`
- 範囲クエリ: `O(log n)`

シャーディング文脈での用途:

- shard ごとの QPS / p95 / queue depth の動的更新
- 一部範囲から最小負荷 shard を選ぶ
- テレメトリ駆動のオンライン再分配

### 5.4 使い分け基準

- ほぼ静的メタデータ + 高クエリ頻度 -> Sparse Table
- 高頻度更新メタデータ -> Segment Tree

要点は、シャーディングの安定性は「どのデータ構造で制御判断を回すか」に強く依存することです。

---

## 6) Fan-out、ホットスポット、再シャーディング

### 6.1 Fan-out は隠れコスト

1 クエリが多数シャードへ広がると:

- tail latency が悪化
- メモリ/マージコストが増加
- 部分障害の影響確率が上昇

低減策:

- 主要フィルタとシャードキーを一致させる
- 時間窓を絞る
- tenant/partition ヒントでルーティングする
- 二段階クエリ（候補抽出 -> 対象取得）

### 6.2 ホットスポット対策

- 書き込み偏り: key salting, adaptive split, buffering
- 読み取り偏り: coalescing, replicas, cache tiers
- テナント偏り: shuffle sharding, quota, fairness scheduler

### 6.3 安全な再シャーディング

1. dual-write + old read
2. 新配置へ backfill
3. shadow read で比較
4. read を切替
5. 検証後に旧配置廃止

シャードキー変更を即時対応で済ませないことが重要です。

---

## 7) データプレーンと制御プレーン

データプレーンはユーザートラフィック処理。
制御プレーンは所有権、リング更新、ヘルス伝播、再配置ループ処理。

重大障害の多くは、ハッシュ関数そのものではなく制御プレーン不安定化で発生します。

### Cortex のリングループ最適化

- PR #7266: **2026年2月16日**にマージ
  - [cortexproject/cortex#7266](https://github.com/cortexproject/cortex/pull/7266)
  - DynamoDB watch loop の `time.After(...)` を再利用 `time.Timer` へ置換
  - PR 掲載ベンチ: 約 `248 B/op, 3 allocs/op` -> `0 B/op, 0 allocs/op`
- PR #7270: **2026年2月20日**にマージ
  - [cortexproject/cortex#7270](https://github.com/cortexproject/cortex/pull/7270)
  - lifecycler / backoff ループまで timer 再利用を拡張

シャーディングへの意味:

制御ループは常時動作するため、微小な allocation でも GC ジッタを増幅し、所有権収束遅延や遷移不安定化を招きます。

---

## 8) 実システムへの対応づけ

### 8.1 Elasticsearch

- index を primary shards + replicas へ分割
- `_id`（または custom key）ハッシュで routing
- 検索は fan-out 後にマージ

設計示唆:

- 局所性が必要なら routing key を設計
- 時間範囲と index 対象を絞って fan-out 抑制
- shard サイズと lifecycle を初期段階で決定

### 8.2 Cortex

- series labels を consistent-hash ring に投入
- ring から replication set を選択
- ring 収束品質が ingestion/query 安定性に直結

設計示唆:

- ring backend レイテンシは所有権伝播に影響
- rollout 時の token movement を制御
- 制御ループ効率がシャード安定性を左右

---

## 9) Go 実装リファレンス

実行可能なサンプルを追加しています:

- `infra/assets/sharding/main.go`

含まれる内容:

1. modulo hash router
2. 仮想ノード付き consistent hash ring + 複製セット取得
3. tenant-aware routing
4. Sparse Table による静的 range-min クエリ
5. Segment Tree による動的 range-min クエリ

実行:

```bash
go run infra/assets/sharding/main.go
```

---

## 10) 最終設計ルール

1. シャードキーはスキーマ都合ではなく主要クエリで決める。
2. 本番前に再シャーディング経路を設計する。
3. shard 単位メトリクス（QPS, p95/p99, queue depth, fan-out）を常時監視する。
4. 制御ループは低 allocation で予測可能に保つ。
5. レプリケーションとトポロジ考慮は必須。
6. Sparse Table / Segment Tree は更新特性で使い分ける。

---

## 参考リンク

- Elasticsearch routing / shard:
  - [Routing field](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-routing-field.html)
  - [Clusters, nodes, and shards](https://www.elastic.co/docs/deploy-manage/distributed-architecture/clusters-nodes-shards)
- Cortex hash ring:
  - [Hash rings](https://cortexmetrics.io/docs/configuration/arguments/#hash-rings)
- Cortex PR:
  - [PR #7266](https://github.com/cortexproject/cortex/pull/7266)
  - [PR #7270](https://github.com/cortexproject/cortex/pull/7270)
