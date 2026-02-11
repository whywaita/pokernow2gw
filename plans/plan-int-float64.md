# Plan: 金額フィールドを int から float64 に移行

## Context

OHH公式仕様フォーマットでは `small_blind_amount: 0.5` のように小数の金額を持つが、内部構造体がすべて `int` 型のため `int()` キャストで切り捨てが発生している。結果として `($0/$1 USD)` と出力されるべきところが `$0.50/$1` にならない。

**フォーマット方針**: 整数はそのまま (`1` → `"1"`)、小数は2桁 (`0.5` → `"0.50"`)

## 変更手順

### 1. 内部構造体の型変更

**`pkg/pokernow2gw/types.go`** — 6フィールドを `int` → `float64`
- `Hand.SmallBlind`, `Hand.BigBlind`, `Hand.Ante`
- `Player.Stack`
- `Action.Amount`
- `Winner.Amount`

### 2. 簡易OHH構造体の型変更

**`pkg/pokernow2gw/ohh_types.go`** — 6フィールドを `int` → `float64`
- `OHHBlinds.SmallBlind`, `OHHBlinds.BigBlind`
- `OHHHand.Ante`
- `OHHPlayer.Stack`
- `OHHAction.Amount`
- `OHHWinner.Amount`

※ OHHSpec側（公式仕様）は既に `float64` なので変更不要

### 3. ヘルパー関数の更新

**`pkg/pokernow2gw/helpers.go`**
- `calculateTotalPot(hand Hand) int` → 戻り値 `float64`
- `calculateRake(totalPot int, bigBlind int, ...) int` → 引数・戻り値 `float64`、`int(rake)` キャスト削除

### 4. フォーマッタの更新

**`pkg/pokernow2gw/formatter.go`**
- `formatNumber(amount float64) string` を新規追加（整数→`"1"`, 小数→`"0.50"`）
  - `math.Trunc(amount) == amount` で判定、`strconv.FormatFloat` で出力
- `formatAmount(amount int, ...) string` → `formatAmount(amount float64, ...) string`
- ヘッダの `$%d/$%d` → `$%s/$%s` + `formatNumber` 呼び出し（80行目, 84行目）
- スタック表示の `$%d` / `%d` → `$%s` / `%s` + `formatNumber`（108行目, 111行目）
- `playerBets map[string]int` → `map[string]float64`、`currentBet` も `float64`
- トーナメントのraise `%d` → `%s` + `formatNumber`（279行目, 281行目）
- `wonAmount` 変数の型を `float64` に

### 5. OHHリーダーの int() キャスト削除

**`pkg/pokernow2gw/ohh_reader.go`** — `convertOHHSpecToHand` 内の6箇所
- `int(p.StartingStack)` → `p.StartingStack`
- `int(a.Amount)` → `a.Amount`
- `int(win.WinAmount)` → `win.WinAmount`
- `int(spec.SmallBlindAmount)` → `spec.SmallBlindAmount`
- `int(spec.BigBlindAmount)` → `spec.BigBlindAmount`
- `int(spec.AnteAmount)` → `spec.AnteAmount`

### 6. CSVパーサーの更新

**`pkg/pokernow2gw/parser.go`** — `strconv.Atoi` 結果を `float64()` でラップ（約15箇所）
- 例: `hand.Ante = amount` → `hand.Ante = float64(amount)`
- 例: `Amount: amount` → `Amount: float64(amount)`
- `parsePlayerStacks` 内の `Stack: stack` → `Stack: float64(stack)`

### 7. テストの更新

**`pkg/pokernow2gw/helpers_test.go`**
- `calculateRake` テスト: 型を `float64` に、小ポットケース (`totalPot=10, 5%`) の期待値 `0` → `0.5`
- `formatNumber` のテストを新規追加

**`pkg/pokernow2gw/ohh_reader_test.go`**
- `TestReadOHHSpec_TableNameAndSiteName`: `($0/$1 USD)` → `($0.50/$1 USD)`

**`pkg/pokernow2gw/converter_test.go`**
- Go の untyped constant ルールにより、数値リテラルは自動的に `float64` に推論される。最小限の変更で済む

## 検証

```bash
go test ./...
go run cmd/pokernow2gw/main.go -cash -rake-cap-bb 4.0 -rake-percent 5.0 --hero-name "adN***" -i ./sample/input/sample_ohh_tenfour.json
```

期待される出力:
```
PokerStars Hand #1: Hold'em No Limit ($0.50/$1 USD) - 2026/02/02 20:17:51 ET
Table 'Ten-Four - ff-26941' 4-max Seat #3 is the button
...
adN***: posts small blind $0.50
IsZ***: posts big blind $1
```
