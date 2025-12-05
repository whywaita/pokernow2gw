# PokerNow MTT → GTO Wizard HH Converter

Go Library + CLI

---

## 1. Overview

PokerNow（pokernow.club）の **MTT ログ（CSV形式）** を、
GTO Wizard が読み込める **GTO Wizard ハンドヒストリー（HH）形式**に変換する
**Go ライブラリ + CLI ツール**。

* ライブラリが主役（変換ロジックを集中）
* CLI はラッパー
* OSS として公開する前提

---

## 2. Goals

* PokerNow の MTT ログを **正しくハンド単位に解析**できること
* GTO Wizard HH 形式に **最小限準拠（GTO Wizard が読み込めるレベル）**
* Hero（解析対象プレイヤー）指定に対応
* CLI から簡単に実行できる
* 将来の GUI/Web 対応は **TODO**

---

## 3. Non-Goals

初期版では以下に対応しない：

* キャッシュゲーム（Ring Game）
* PLO・Mixed Games など Hold’em 以外
* 正確な役名テキスト生成
* 細かいサイドポット計算の完全一致
* Buy-in/Currency を含む詳細ヘッダ
* サーバー運用や保存

---

## 4. Input Format (PokerNow CSV)

PokerNow の「ログをダウンロード」で得られる CSV。

列定義：

| column  | description                        |
| ------- | ---------------------------------- |
| `entry` | 1行分のログ（ハンド開始、スタック、アクション、ショウダウン、終了） |
| `at`    | ISO8601 UTC タイムスタンプ                |
| `order` | 並び順キー（昇順で処理）                       |

例：

```
-- starting hand #120 (id: rnqj5zrnvdqx)  (No Limit Texas Hold'em) (dealer: "ramunen @ ...") --
Player stacks: #5 "ramunen @ ..." (66998) | #9 "whywaita @ ..." (383002)
"whywaita @ ..." posts a small blind of 2000
...
-- ending hand #120 --
```

---

## 5. Output Format (GTO Wizard HH)

* 1ファイルに複数ハンド連結
* GTO Wizard の HH に準拠
* GTO Wizard が読み込める最低限の情報のみ必須

### 必須出力要素

* Hand Header
* Table Info / Seat Info
* Blinds / Antes
* Actions: Preflop / Flop / Turn / River
* Board Cards
* Showdown
* Pot Collection（最低限）

---

## 6. High-Level Architecture

```
CLI → (CSV Reader) → []LogEntry → ConvertEntries() → ConvertResult → Output
```

ロジックはすべてライブラリ内部。

---

## 7. Library API Specification

```go
type LogEntry struct {
    Entry string
    At    time.Time
    Order int64
}

type ConvertOptions struct {
    HeroName       string        // 表示名（例: "whywaita"）
    SiteName       string        // "GTO Wizard" (default)
    TimeLocation   *time.Location// "UTC", "Asia/Tokyo" 等
    TournamentName string        // optional
}

type ConvertResult struct {
    HH           []byte // GTO Wizard HH text
    SkippedHands int    // パースに失敗したハンド数
}

// CSV から読み込んで変換
func ConvertCSV(r io.Reader, opts ConvertOptions) (*ConvertResult, error)

// すでに構築済み LogEntry スライスから変換
func ConvertEntries(entries []LogEntry, opts ConvertOptions) (*ConvertResult, error)
```

---

## 8. CLI Specification

コマンド名（例）：`pokernow2gw`

### Usage

```
pokernow2gw \
  --input pokernow.csv \
  --output out_hh.txt \
  --hero-name "whywaita" \
  --site-name "GTO Wizard" \
  --timezone "UTC"
```

### Options

| option              | description                     |
| ------------------- | ------------------------------- |
| `--input, -i`       | 入力CSV                           |
| `--output, -o`      | 出力ファイル（省略時は stdout）             |
| `--hero-name`       | Hero 表示名（例: `whywaita`）         |
| `--site-name`       | HH のサイト名（デフォルト: GTO Wizard）     |
| `--timezone`        | 出力HHのタイムゾーン（例: UTC, Asia/Tokyo） |
| `--tournament-name` | 任意のタイトル（省略可）                    |

### Behavior

* 変換後 `X hands were skipped due to parse errors.` を stderr に出力
* 致命的なパースエラーのみ exit code ≠ 0

---

## 9. Conversion Rules (PokerNow → GTO Wizard)

### 9.1 Hand Detection

* 開始：`-- starting hand #N (id: X) ...`
* 終了：`-- ending hand #N --`
* 1ハンド単位で処理する。
* エラーが発生したハンドは **スキップ**し、`SkippedHands++`。

### 9.2 Player Name Mapping

PokerNow 形式：

```
"whywaita @ DtjzvbAuKs"
```

表示名（Hero 用）は：

```
whywaita
```

抽出方法：

1. `"..."` 内の文字列を取得
2. `"@"` で split し、左側（trim済み）を表示名とする

### 9.3 Tournament / Table Info

* トーナメント ID：
  → starting hand 行の `(id: <id>)` の `<id>` をそのまま利用
* Table 名：
  → `Table 'PokerNow <id>' N-max`

  * `N` はそのハンドのプレイヤー人数

### 9.4 Player Stacks

`Player stacks:` から以下を取得：

* Seat 番号（`#5` → 5）
* 表示名
* スタック量

例：

```
Seat 5: ramunen (66998)
Seat 9: whywaita (383002)
```

### 9.5 Blinds / Antes

以下の行に対応：

```
posts a small blind of X
posts a big blind of Y
posts an ante of Z
```

### 9.6 Actions

対応パターン例：

| PokerNow                        | GTO Wizard                     |
| ------------------------------- | ------------------------------ |
| `"X" folds`                     | `X: folds`                     |
| `"X" checks`                    | `X: checks`                    |
| `"X" calls N`                   | `X: calls N`                   |
| `"X" bets N`                    | `X: bets N`                    |
| `"X" raises to N`               | `X: raises to N`               |
| `"X" raises to N and go all in` | `X: raises to N and is all-in` |

### 9.7 Streets

* Preflop：開始〜`Flop:` の前
* Flop：`Flop:` 行〜`Turn:` の前
* Turn：`Turn:` 行〜`River:` の前
* River：`River:` 行〜ショウダウン前

### 9.8 Board Cards

変換ルール：

| PokerNow | PS   |
| -------- | ---- |
| `A♥`     | `Ah` |
| `4♦`     | `4d` |

### 9.9 Showdown

* `"X" shows a A♥, 4♦.`
  → `X: shows [Ah 4d]`
* `"X" collected N from pot`
  → `X collected N from pot`

役名テキストは省略（TODO）。

---

## 10. Error Handling

* ハンド単位のパース失敗は **スキップ**
* すべてのエラーを集計して `SkippedHands` として返す
* CLI は終了時に `X hands were skipped` を表示

---

## 11. Timezone Handling

* CSV `at` は UTC (`Z`)
* HH 出力は `ConvertOptions.TimeLocation` に従って変換

  * `--timezone` で指定

---

## 12. Extensibility (TODO)

* Web UI / GUI 対応
* 役名の自動生成
* 正確なポット・サイドポット計算
* Buy-in / Currency のヘッダ追加
* Ring game / PLO 対応
* GG / PartyPoker 等への変換

---

## 13. License

OSS として公開
MIT

---

## 14. Appendix: Development Notes

* Golang 最新
* 標準 CSV パーサを利用
* 変換部分は関数を分割しテストしやすい構造にする
* Goldentest による HH 出力の固定比較を推奨
