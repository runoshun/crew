# ダイアログ背景色 PoC メモ

## 背景

lipgloss の `Place` を使ってダイアログを中央寄せすると、`Place` が挿入する余白に背景色が付かず、ダイアログ外周にベース背景が露出する問題が再現できた。`dialogStyle()` の `Background` だけでは「既に存在する文字」にしかスタイルが当たらないため、中央寄せで生じる左右・上下のスペースが無着色のまま残る。

## 実験結果

`internal/tui/poc_test.go` では以下を確認している。

1. `lipgloss.Place` 単体では先頭・末尾の余白に ANSI シーケンスが付かず、文字列プレフィクスは単なる `"   "` になる。
2. `lipgloss.WithWhitespaceBackground(bg)` と `lipgloss.WithWhitespaceChars(" ")` を `Place` に渡すと、余白部分にも `\x1b[48;2;...m` が付き、改行で増えた上下のブランク行も同じ背景色で埋まる。
3. `lipgloss.Place(..., height>1, ...)` にも同オプションを渡すことで、縦方向のパディング行も着色される。

## 推奨アプローチ

1. `overlayDialog` などで `lipgloss.Place` を呼ぶ箇所に以下のオプションを追加する。

   ```go
   lipgloss.Place(
       width,
       height,
       lipgloss.Center,
       lipgloss.Center,
       content,
       lipgloss.WithWhitespaceBackground(Colors.Background),
       lipgloss.WithWhitespaceChars(" "),
   )
   ```

   - `WithWhitespaceBackground` が `Place` の内部空白レンダラに背景色を伝搬させる。
   - `WithWhitespaceChars(" ")` を指定しておくと、Wide 文字を避けつつ安定した塗り潰しができる。

2. 背景色を変えたい場合は `Colors.DialogBackground` のような専用値を設け、このオプションにも同じ色を渡す。

3. 余白の塗り潰しを複数箇所で再利用するために、以下のようなヘルパーを追加すると分かりやすい。

   ```go
   func placeDialog(width, height int, bg lipgloss.Color, content string) string {
       return lipgloss.Place(
           width,
           height,
           lipgloss.Center,
           lipgloss.Center,
           content,
           lipgloss.WithWhitespaceBackground(bg),
           lipgloss.WithWhitespaceChars(" "),
       )
   }
   ```

## 次アクション案

- `overlayDialog` に上記ヘルパーを適用し、特に `m.width-appPadding` × `m.height-2` の余白を塗り潰す。
- モーダル表示以外でも `lipgloss.Place` を利用している箇所があれば同じオプションを共有する。
