# E2E Test Guide

手動E2Eテストの手順書。自動テストでカバーできない範囲を検証する。

## 前提条件

- `mise run build` でバイナリがビルド済み
- 各CLI（claude, opencode, codex）がインストール済み

---

## 環境セットアップ

### 自動セットアップ

```bash
# テスト用リポジトリを作成（.e2e-test/ に作成、gitignored）
./scripts/e2e-setup.sh

# または任意のディレクトリに作成
./scripts/e2e-setup.sh /path/to/test-dir
```

### 手動セットアップ

```bash
# 1. テスト用ディレクトリ作成（リポジトリ内、gitignored）
TEST_DIR=.e2e-test
mkdir -p $TEST_DIR && cd $TEST_DIR

# 2. gitリポジトリ初期化
git init
git config user.email "test@example.com"
git config user.name "E2E Test"

# 3. 初期ファイル作成
echo "# E2E Test" > README.md
git add . && git commit -m "Initial commit"

# 4. crew初期化
crew init
```

### クリーンアップ

```bash
# テスト完了後
rm -rf .e2e-test
```

**Note:** `.e2e-test/` はメインリポジトリの `.gitignore` に追加済み。エージェントの権限設定上、リポジトリ内に配置することで worktree アクセスが容易になる。

---

## 1. ステータス遷移テスト

各エージェントがステータスを正しく遷移させるか検証する。

### 1.1 Claude (cc-medium) ステータス遷移

**期待動作:**
- 起動時: `todo` → `in_progress`
- アイドル/パーミッション要求時: `in_progress` → `needs_input`
- ユーザー入力後: `needs_input` → `in_progress`
- `crew complete` 実行後: `in_progress` → `in_review`

**手順:**

```bash
# 1. テストタスク作成
crew new --title "E2E: Claude ステータス遷移テスト"

# 2. タスク開始
crew start <id> cc-medium

# 3. ステータス確認（in_progress になるはず）
crew list | grep <id>

# 4. エージェントがアイドルになるまで待機
sleep 30

# 5. ステータス確認（needs_input になるはず）
crew list | grep <id>

# 6. 何か入力を送信
crew send <id> "echo test"
crew send <id> Enter

# 7. ステータス確認（in_progress に戻るはず）
sleep 5
crew list | grep <id>

# 8. 完了指示
crew send <id> "crew complete <id>"
crew send <id> Enter
# パーミッション許可
crew send <id> y
crew send <id> Enter

# 9. ステータス確認（in_review になるはず）
sleep 10
crew list | grep <id>

# 10. クリーンアップ
crew close <id>
```

**検証ポイント:**
- [ ] 起動後 `in_progress` に遷移
- [ ] アイドル時 `needs_input` に遷移
- [ ] 入力後 `in_progress` に戻る
- [ ] complete 後 `in_review` に遷移

---

### 1.2 OpenCode (oc-medium-ag) ステータス遷移

**期待動作:** Claude と同様

**手順:**

```bash
# 1. テストタスク作成
crew new --title "E2E: OpenCode ステータス遷移テスト"

# 2. タスク開始
crew start <id> oc-medium-ag

# 3-9. Claude と同様の手順で検証

# 10. クリーンアップ
crew close <id>
```

**検証ポイント:**
- [ ] TypeScript plugin が正しくロードされる
- [ ] 各イベントでステータスが遷移する

---

### 1.3 Codex ステータス遷移

**期待動作:**
- `agent-turn-complete` イベント時に `in_progress` → `needs_input`

**手順:**

```bash
# 1. テストタスク作成
crew new --title "E2E: Codex ステータス遷移テスト"

# 2. タスク開始
crew start <id> codex

# 3. ステータス確認
crew list | grep <id>

# 4. エージェントがターンを完了するまで待機（プロンプト表示など）
# codex は notify が agent-turn-complete のみなので、
# アイドル検知は限定的

# 5. ステータス確認
crew list | grep <id>

# 6. クリーンアップ
crew close <id>
```

**検証ポイント:**
- [ ] notify 設定が正しくパースされる
- [ ] ターン完了時にステータス遷移する

**既知の制限:**
- codex の notify は `agent-turn-complete` のみ対応
- アイドル状態の即座な検知は不可

---

## 2. TUI操作テスト

### 2.1 基本操作

**手順:**

```bash
# 1. TUI起動
crew

# 2. 操作確認
# - j/k: タスク選択
# - Enter: 詳細表示
# - s: タスク開始
# - p: peek（セッション出力確認）
# - a: attach（セッションにアタッチ）
# - d: diff表示
# - ?: ヘルプ表示
# - q: 終了
```

**検証ポイント:**
- [ ] キーバインドが正しく動作
- [ ] ステータス表示がリアルタイム更新
- [ ] エラー時の表示が適切

---

### 2.2 peek/attach

**手順:**

```bash
# 1. タスク開始
crew start <id> cc-small

# 2. TUIで peek
crew
# p キーで peek

# 3. 出力が表示されることを確認

# 4. attach
# a キーで attach
# Ctrl+b d で detach
```

**検証ポイント:**
- [ ] peek で最新出力が表示
- [ ] attach でセッションに接続
- [ ] detach で TUI に戻る

---

### 2.3 send

**手順:**

```bash
# 1. タスク開始（needs_input 状態で）
crew start <id> cc-small

# 2. needs_input になるまで待機
sleep 30

# 3. キー送信
crew send <id> "ls -la"
crew send <id> Enter

# 4. peek で確認
crew peek <id>
```

**検証ポイント:**
- [ ] 文字列が正しく送信される
- [ ] Enter が送信される
- [ ] 特殊キー（Ctrl+C など）が送信される

---

## 3. ワークフローテスト

### 3.1 タスク完了フロー

```bash
# 1. タスク作成
crew new --title "E2E: 完了フローテスト" --body "echo hello を実行してください"

# 2. 開始
crew start <id> cc-small

# 3. 完了まで待機（または手動で crew complete）

# 4. レビュー
crew review <id>

# 5. マージ
echo "y" | crew merge <id>

# 6. 確認
crew show <id>  # status: done
```

**検証ポイント:**
- [ ] todo → in_progress → in_review → done の遷移
- [ ] worktree が削除される
- [ ] main にマージされる

---

### 3.2 レビュー＆修正フロー

```bash
# 1. タスク作成・開始・完了
crew new --title "E2E: レビュー修正フローテスト"
crew start <id> cc-small
# ... in_review まで進める

# 2. レビュー（修正要求）
crew review <id>

# 3. コメント送信
crew comment <id> -R "修正してください: XXX"

# 4. ステータス確認（in_progress に戻るはず）
crew list | grep <id>

# 5. 再度 in_review まで進める

# 6. 再レビュー・マージ
crew review <id>
echo "y" | crew merge <id>
```

**検証ポイント:**
- [ ] comment -R で in_progress に戻る
- [ ] エージェントがコメントを読んで修正する

---

## 4. エラーケーステスト

### 4.1 存在しないタスク

```bash
crew show 99999
# エラーメッセージが適切か
```

### 4.2 存在しないエージェント

```bash
crew start <id> nonexistent-agent
# エラーメッセージが適切か
```

### 4.3 既に開始済みタスク

```bash
crew start <id> cc-small
crew start <id> cc-small  # 2回目
# エラーまたは適切な動作か
```

---

## 5. poll コマンドテスト

```bash
# 1. タスク作成・開始
crew new --title "E2E: poll テスト"
crew start <id> cc-small

# 2. バックグラウンドで poll
crew poll <id> --timeout 120 &

# 3. ステータス変化を確認
# needs_input になったら出力されるはず

# 4. 確認後 poll を停止
kill %1
```

**検証ポイント:**
- [ ] ステータス変化時に出力
- [ ] タイムアウトで終了
- [ ] 終端状態で終了

---

## チェックリスト

### ステータス遷移
- [ ] Claude: in_progress → needs_input → in_progress → in_review
- [ ] OpenCode: 同上
- [ ] Codex: notify によるステータス変更

### TUI
- [ ] 基本操作（j/k/Enter/q）
- [ ] peek/attach/detach
- [ ] send

### ワークフロー
- [ ] 完了フロー（todo → done）
- [ ] レビュー修正フロー

### エラーケース
- [ ] 存在しないリソースへのアクセス
- [ ] 重複操作

### poll
- [ ] ステータス変化検知
- [ ] タイムアウト・終了条件
