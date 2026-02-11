# Fix: OHH spec形式でヒーローカードが誤ったプレイヤーから取得される

## Context

OHH spec形式のJSON読み込み時、ヒーローカードの特定に `hero_player_id`（JSON内）のみを使用しており、`--hero-name`（CLI引数）が無視されている。

サンプルJSON (`sample_ohh_tenfour.json`) では:
- `hero_player_id: 1` → IsZ*** (cards: `["Ks", "9s"]`)
- adN*** は id=4 (cards: `["Ts", "Jh"]` = JTo)

`--hero-name "adN***"` を指定しても、IsZ***のカード (Ks 9s) がヒーローカードとして出力される。

**根本原因**: `convertOHHSpecToHand` (`ohh_reader.go:152-158`) が `opts.HeroName` を一切参照していない。

## 変更内容

### 1. `convertOHHSpecToHand` のヒーローカード取得ロジックを修正

**`pkg/pokernow2gw/ohh_reader.go`** (行 152-158)

現在:
```go
var heroCards []string
if spec.HeroPlayerID > 0 {
    if heroPlayer, ok := playerMap[spec.HeroPlayerID]; ok {
        heroCards = heroPlayer.Cards
    }
}
```

修正後（`opts.HeroName` を優先、`hero_player_id` はフォールバック）:
```go
var heroCards []string
if opts.HeroName != "" {
    for _, p := range spec.Players {
        if p.Name == opts.HeroName {
            heroCards = p.Cards
            break
        }
    }
}
if len(heroCards) == 0 && spec.HeroPlayerID > 0 {
    if heroPlayer, ok := playerMap[spec.HeroPlayerID]; ok {
        heroCards = heroPlayer.Cards
    }
}
```

### 2. テスト追加

**`pkg/pokernow2gw/ohh_reader_test.go`** に `TestConvertOHHSpecToHand_HeroCardsByName` を追加

| ケース | heroName | 期待されるカード |
|--------|----------|------------------|
| HeroNameが別プレイヤーを指す（バグ再現） | `"Bob"` | Bob のカード |
| HeroNameがhero_player_idと一致（後方互換） | `"Alice"` | Alice のカード |
| HeroNameが見つからない（フォールバック） | `"Charlie"` | hero_player_id のカード |
| HeroNameが空（フォールバック） | `""` | hero_player_id のカード |

## 変更対象ファイル

| ファイル | 変更内容 |
|----------|----------|
| `pkg/pokernow2gw/ohh_reader.go` | 行 152-158: 名前ベースのヒーローカード検索を追加 |
| `pkg/pokernow2gw/ohh_reader_test.go` | テスト関数を1つ追加（4ケース） |

## 検証

```bash
go test ./pkg/pokernow2gw/ -run TestConvertOHHSpecToHand_HeroCardsByName -v
go test ./...
go run cmd/pokernow2gw/main.go -cash -rake-cap-bb 4.0 -rake-percent 5.0 --hero-name "adN***" -i ./sample/input/sample_ohh_tenfour.json
```

期待出力: `Dealt to adN*** [Ts Jh]`（現在は誤って `[Ks 9s]`）
