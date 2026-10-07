# Go言語におけるメッセージキューパターン

> **⏱️ 所要時間**: 全パターンで約 45 分（各パターンは単独でも読めます）
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.22+、OS 不問
>
> **📖 参照カタログ**: 本書はステップバイステップのハンズオンではなく、リファレンス型（カタログ）です。各パターンは独立した内容なので、必要なパターンだけをピックアップして読んでください。各パターンのセクションでは、対応アセットからそのまま抜粋した核心コードと、実行可能なリファレンス実装のパスを併記します。

[English](go_mq_patterns_en.md) | [日本語](go_mq_patterns_ja.md)

## 目次

- [このカタログの使い方](#このカタログの使い方)
- [メッセージキューシステムとツールの理解](#メッセージキューシステムとツールの理解)
- [パターン1: シンプルなチャネルベースキュー](#パターン1-シンプルなチャネルベースキュー)
- [パターン2: ファンアウトメッセージキュー](#パターン2-ファンアウトメッセージキュー)
- [パターン3: ワーカープールパターン](#パターン3-ワーカープールパターン)
- [パターン4: 分散Kafkaキュー](#パターン4-分散kafkaキュー)
- [パターン5: リトライとデッドレターキュー](#パターン5-リトライとデッドレターキュー)

---

## このカタログの使い方

本書は**リファレンス型（カタログ）**であり、ハンズオン形式ではありません。セットアップすべき環境もビルドするものもありません。各パターンのセクションでは概念を解説した後、対応アセットからそのまま抜粋した核心コードと、完全な実装のパスを示します。

実際のブローカーを使ったハンズオンは [RabbitMQ 実習](../infra/rabbitmq_ja.md) を参照してください。キャッシュ側のカタログは [Go言語におけるキャッシュパターン](go_cache_patterns_ja.md) です。

### 対応アセット

| パターン                          | リファレンス実装                                                                         |
| :-------------------------------- | :--------------------------------------------------------------------------------------- |
| 1. シンプルなチャネルベースキュー | [pattern1_simple_channel/main.go](assets/go_mq_patterns/pattern1_simple_channel/main.go) |
| 2. ファンアウトメッセージキュー   | [pattern2_fanout/main.go](assets/go_mq_patterns/pattern2_fanout/main.go)                 |
| 3. ワーカープールパターン         | [pattern3_worker_pool/main.go](assets/go_mq_patterns/pattern3_worker_pool/main.go)       |
| 4. 分散Kafkaキュー                | [pattern4_kafka/main.go](assets/go_mq_patterns/pattern4_kafka/main.go)                   |
| 5. リトライとデッドレターキュー   | [pattern5_retry_dlq/main.go](assets/go_mq_patterns/pattern5_retry_dlq/main.go)           |

---

## メッセージキューシステムとツールの理解

### 基礎

#### メッセージキューとは？

メッセージキューは、サービスやコンポーネント間で非同期にデータを交換するための通信メカニズムです。
主な目的は、**プロデューサーとコンシューマーを分離**し、**信頼性のあるメッセージ配信を保証**することです。

**例え**
受取人がすぐに開けるのを待たずに手紙を投函できる郵便ポストのようなものです。

---

#### なぜメッセージキューが重要なのか：システムへのメリット

##### 疎結合アーキテクチャ

- サービスを独立して進化させられる
- 障害がコンポーネント間で連鎖しない
- 個々のサービスをスケールしやすい

##### 負荷平準化

- トラフィックスパイクを吸収
- バックエンドの過負荷を防止
- 処理レートを平滑化

##### 信頼性

- 処理されるまでメッセージが永続化
- 配信保証のセマンティクス
- 障害時のリトライメカニズム

**実例**
EC プラットフォームでは、ピーク時のトラフィックでも注文が失われないよう、メッセージキューを使用して非同期に注文を処理しています。

---

#### メッセージキューアーキテクチャ：コンポーネント

##### プロデューサー

- メッセージを作成して送信
- ファイア・アンド・フォーゲット型のパブリッシュ
- 効率化のためにメッセージをバッチ処理可能

##### キュー / トピック

- メッセージを一時的に保存
- 順序を維持（FIFO またはパーティション）
- 永続性の保証を提供

##### コンシューマー

- メッセージを受信して処理
- 処理成功を確認応答
- 水平スケール可能

---

#### メッセージングパターン：メッセージの流れ

##### 1. ポイント・ツー・ポイント（キュー）

- 1 プロデューサー、メッセージごとに 1 コンシューマー
- 各メッセージは正確に 1 回配信
- ワーカー間での作業分配

##### 2. パブリッシュ・サブスクライブ（トピック）

- 1 プロデューサー、複数コンシューマー
- 各サブスクライバーがすべてのメッセージを受信
- イベントのブロードキャストと通知

##### 3. リクエスト・リプライ

- 非同期トランスポート上での同期的な通信
- 相関 ID でリクエストとレスポンスを紐付け
- RPC スタイルのインタラクションに有用

---

#### 配信保証：信頼性のスペクトラム

> 「Exactly-once配信はメッセージングシステムの聖杯である」

##### At-Most-Once（最大1回）

- メッセージが失われる可能性がある
- 障害時のリトライなし
- 最低レイテンシ

##### At-Least-Once（最低1回）

- メッセージが 1 回以上配信される
- 冪等なコンシューマーが必要
- 実務で最も一般的

##### Exactly-Once（正確に1回）

- メッセージが正確に 1 回配信される
- トランザクションサポートが必要
- 最高の複雑性とオーバーヘッド

---

#### 注目のツール：Apache Kafka

Kafka は分散イベントストリーミングプラットフォームで、一般的に以下として使用されます：

- メッセージキュー
- イベントストア
- ストリームプロセッサー

高スループットシステムで広く採用されています。

##### Kafkaの主な特徴

- 高スループット（毎秒数百万メッセージ）
- 分散・耐障害性
- 永続的なメッセージストレージ
- 並列処理のためのコンシューマーグループ
- 設定可能な保持ポリシー

##### 一般的なユースケース

###### イベントソーシング

- すべての変更をイベントとしてキャプチャ
- イベントログから状態を再構築

###### ログ集約

- 複数サービスからログを収集
- 一元的な処理と分析

###### ストリーム処理

- リアルタイムデータ変換
- ウィンドウ集計
- 複合イベント処理

---

#### 判断ガイド：メッセージキューを使用すべき場合

##### 理想的なユースケース

- 非同期ワークフロー
- マイクロサービス間通信
- バックグラウンドジョブ処理
- イベント駆動アーキテクチャ

##### 適さない場合

- リアルタイム・低レイテンシ要件
- シンプルなリクエスト・レスポンス API
- 小規模なモノリシックアプリケーション

##### 考慮すべきトレードオフ

- インフラの複雑性が増加
- メッセージ順序の課題
- 分散フローのデバッグ

---

#### まとめ：統合基盤としてのメッセージキュー

1. **キューは疎結合を実現**
   サービスは密結合なしに通信できる。

2. **適切なパターンを選択**
   ニーズに応じてポイント・ツー・ポイント、pub-sub、リクエスト・リプライを選択。

3. **Kafkaはスケールを提供**
   高スループット分散システムに本番対応。

4. **障害を優雅に処理**
   リトライ、デッドレターキュー、モニタリングを実装。

---

## パターン1: シンプルなチャネルベースキュー

### コアコンセプト

Go ネイティブのチャネルを使用したインメモリメッセージパッシングの基本パターン。
Go の並行性プリミティブを活用し、設計上スレッドセーフです。

### チャネルキューフロー図

```mermaid
flowchart TD
    P[プロデューサー] --> Q[チャネルキュー]
    Q --> C[コンシューマー]
    P -->|Publish| Q
    Q -->|Subscribe| C
    C -->|処理| R[結果]
```

### 主な実装詳細

- 非同期パブリッシュのためのバッファ付きチャネル
- 満杯時にエラーを返すノンブロッキングのパブリッシュ
- チャネルクローズによるクリーンなシャットダウン
- 外部依存なし

### 使用すべき場合

シンプルな非同期通信が必要な**シングルプロセスアプリケーション**に最適。

典型的なユースケース：

- 内部ゴルーチン間の調整
- パイプライン処理
- サービス内でのイベントディスパッチ

### トレードオフ

- 再起動時に永続化されない
- シングルプロセスに限定
- バッファサイズの調整が必要

### 核心コード（抜粋）

[pattern1_simple_channel/main.go](assets/go_mq_patterns/pattern1_simple_channel/main.go) より抜粋 —— ブロックせず、バッファ満杯をエラーとして報告するノンブロッキング publish：

```go
func (q *ChannelQueue) Publish(msg Message) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case q.ch <- msg:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}
```

### リファレンス実装

`software/assets/go_mq_patterns/pattern1_simple_channel/main.go`

---

## パターン2: ファンアウトメッセージキュー

### ファンアウト概要

登録されたすべてのサブスクライバーに各メッセージを配信するブロードキャストパターン。
イベント通知やリアルタイム更新に有用です。

### ファンアウトフロー図

```mermaid
flowchart TD
    P[プロデューサー] --> Q[ファンアウトキュー]
    Q --> S1[サブスクライバー1]
    Q --> S2[サブスクライバー2]
    Q --> S3[サブスクライバー3]
    S1 --> R1[処理]
    S2 --> R2[処理]
    S3 --> R3[処理]
```

### 実装ステップ

1. **サブスクライバーレジストリ**
   - サブスクライバーID からチャネルへのマップ
   - スレッドセーフな登録/登録解除

2. **パブリッシュ時のブロードキャスト**
   - すべてのサブスクライバーを反復
   - 各チャネルにメッセージを送信

3. **遅いサブスクライバーの処理**
   - select によるノンブロッキング送信
   - バッファが一杯の場合はメッセージをドロップ

4. **クリーンなシャットダウン**
   - すべてのサブスクライバーチャネルをクローズ
   - レジストリから削除

### なぜこれが機能するか

ファンアウトにより以下が可能：

- 複数のコンシューマーが同じイベントを処理
- リアルタイム通知
- キャッシュ無効化のブロードキャスト

### 核心コード（抜粋）

[pattern2_fanout/main.go](assets/go_mq_patterns/pattern2_fanout/main.go) より抜粋 —— 遅いサブライバーのメッセージをドロップするノンブロッキング送信によるブロードキャスト：

```go
func (q *FanoutQueue) Publish(msg Message) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, sub := range q.subscribers {
		select {
		case sub.ch <- msg:
		default:
			fmt.Printf("Subscriber %s buffer full, dropping message\n", sub.id)
		}
	}
}
```

### ファンアウトリファレンス実装

`software/assets/go_mq_patterns/pattern2_fanout/main.go`

---

## パターン3: ワーカープールパターン

### 概要

固定数のワーカープールに作業を分配して並列処理。
スループットを最大化しながらリソース使用を制御します。

### ワーカープールフロー図

```mermaid
flowchart TD
    J1[ジョブ1] --> Q[ジョブキュー]
    J2[ジョブ2] --> Q
    J3[ジョブ3] --> Q
    Q --> W1[ワーカー1]
    Q --> W2[ワーカー2]
    Q --> W3[ワーカー3]
    W1 --> R[結果キュー]
    W2 --> R
    W3 --> R
```

### リクエストフロー

1. **ジョブの投入**
   - ジョブチャネルに作業項目をプッシュ

2. **ワーカー処理**
   - 固定数のゴルーチンがチャネルからプル
   - 各ワーカーが独立して処理

3. **結果の収集**
   - ワーカーが出力チャネルに結果をプッシュ
   - アグリゲーターが収集して処理

4. **グレースフルシャットダウン**
   - ジョブチャネルをクローズ
   - ワーカーの完了を待機

### なぜワーカープールが重要か

#### プールなしの場合

- 無制限なゴルーチン作成
- メモリ枯渇のリスク
- バックプレッシャーなし

#### プールありの場合

- 制御された並行性
- 予測可能なリソース使用
- チャネルブロッキングによる組み込みバックプレッシャー

### 核心コード（抜粋）

[pattern3_worker_pool/main.go](assets/go_mq_patterns/pattern3_worker_pool/main.go) より抜粋 —— 固定数のワーカーがジョブチャネルから取り出し、`Close` でグレースフルにドレイン：

```go
func (p *WorkerPool) Start(processor func(Job) Result) {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			for job := range p.jobs {
				fmt.Printf("Worker %d processing job %s\n", workerID, job.ID)
				result := processor(job)
				p.results <- result
			}
		}(i)
	}
}

func (p *WorkerPool) Submit(job Job) {
	p.jobs <- job
}

func (p *WorkerPool) Results() <-chan Result {
	return p.results
}

func (p *WorkerPool) Close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}
```

### ワーカープールリファレンス実装

`software/assets/go_mq_patterns/pattern3_worker_pool/main.go`

---

## パターン4: 分散Kafkaキュー

### Kafka概要

Kafka は業界標準の分散メッセージングソリューションです。
複数のサービスが共有トピックからプロデュース・コンシュームでき、イベント駆動アーキテクチャを実現します。

### Kafkaフロー図

```mermaid
flowchart TD
    subgraph プロデューサー
        P1[サービスA]
        P2[サービスB]
    end
    subgraph Kafkaクラスター
        T[トピック: orders]
    end
    subgraph コンシューマーグループ
        C1[コンシューマー1]
        C2[コンシューマー2]
    end
    P1 -->|produce| T
    P2 -->|produce| T
    T -->|partition 0| C1
    T -->|partition 1| C2
```

### 主な特性

#### パーティション化されたトピック

- メッセージがパーティション間で分散
- パーティションごとの並列消費
- パーティション内での順序保証

#### コンシューマーグループ

- 複数のコンシューマーがワークロードを共有
- 自動パーティション割り当て
- コンシューマー変更時のリバランシング

#### 永続ストレージ

- メッセージがディスクに永続化
- 設定可能な保持期間
- リプレイ機能

### クライアントライブラリ

`segmentio/kafka-go` が提供する機能：

- イディオマティックな Go API
- 自動パーティションバランシング
- コンシューマーグループ管理
- バッチ操作

### 核心コード（抜粋）

[pattern4_kafka/main.go](assets/go_mq_patterns/pattern4_kafka/main.go) より抜粋 —— プロデューサー設定（バランサー、acks）と 1 回の `WriteMessages` 呼び出し：

```go
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, key, value string) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	})
}
```

### Kafkaリファレンス実装

`software/assets/go_mq_patterns/pattern4_kafka/main.go`

---

## パターン5: リトライとデッドレターキュー

### DLQ概要

リトライロジックとポイズンメッセージ用のデッドレターキューでメッセージ処理障害を処理。
このパターンは**速度より信頼性**を優先します。

### リトライフロー図

```mermaid
flowchart TD
    M[メッセージ] --> Q[メインキュー]
    Q --> W[ワーカー]
    W --> D{成功?}
    D -- yes --> Done[完了]
    D -- no --> R{リトライ残り?}
    R -- yes --> RQ[リトライキュー]
    RQ -->|バックオフ遅延| W
    R -- no --> DLQ[デッドレターキュー]
    DLQ --> Manual[手動レビュー]
```

### 処理フロー

1. **メッセージ受信**
   - メインキューからプル

2. **ハンドラーで処理**
   - ビジネスロジックを実行
   - 試行回数を追跡

3. **障害処理**
   - リトライ回数を最大値と比較

- 指数バックオフを適用（初回は基準遅延、以降は係数で増加）
  - 再キューまたは DLQ へ移動

1. **デッドレターキュー**
   - 永続的に失敗したメッセージを保存
   - 手動検査を可能に
   - 再処理をサポート

### メリット

- 一時的障害でメッセージを失わない
- バックオフ付き自動リトライ
- ポイズンメッセージの隔離
- 障害パターンの可観測性

### DLQのトレードオフ

- 全体的な処理が遅くなる
- DLQ のモニタリングが必要
- 手動介入が必要な場合がある

### 核心コード（抜粋）

[pattern5_retry_dlq/main.go](assets/go_mq_patterns/pattern5_retry_dlq/main.go) より抜粋 —— リトライか DLQ かの判定部。再キューはノンブロッキングにして `Stop()` 中でもデッドロックしないようにしています：

```go
func (q *RetryableQueue) handleMessage(workerID int, msg Message) {
	msg.Attempts++

	if err := q.handler(msg); err != nil {
		if msg.Attempts >= q.config.MaxRetries {
			fmt.Printf("Message %s exceeded max retries, moving to DLQ\n", msg.ID)
			q.dlq.Add(msg)
		} else {
			fmt.Printf("Message %s failed (attempt %d/%d), scheduling retry\n",
				msg.ID, msg.Attempts, q.config.MaxRetries)
			// Never block forever here: if Stop() is running, drop the retry
			// instead of waiting on a queue nobody will drain anymore.
			select {
			case q.retryQueue <- msg:
			case <-q.stopCh:
			}
		}
	} else {
		fmt.Printf("Message %s processed successfully\n", msg.ID)
	}
}
```

### DLQリファレンス実装

`software/assets/go_mq_patterns/pattern5_retry_dlq/main.go`

---

## 学んだこと

- チャネルベースのプロセス内キューで十分なケースと、本物のメッセージブローカーが必要なケースの見分け方
- ファンアウトとワーカープールによるメッセージ消費の水平スケールの方法
- プロセス内キューから分散 Kafka キューへ、プロデューサー/コンシューマーの契約を変えずに移行する方法
- リトライ・バックオフ・デッドレターキューを使って恒久的な障害を安全に処理する方法

**次のステップ**: コンテナ化されたブローカーをエンドツーエンドで構築してみましょう。[RabbitMQ 実習](../infra/rabbitmq_ja.md) を参照してください。

---

## 参考文献

- [Go の並行処理（チャネルの基礎）](https://go.dev/tour/concurrency)
- [Apache Kafka ドキュメント](https://kafka.apache.org/documentation/)
- [segmentio/kafka-go クライアント](https://github.com/segmentio/kafka-go)
- [RabbitMQ チュートリアル](https://www.rabbitmq.com/tutorials)
