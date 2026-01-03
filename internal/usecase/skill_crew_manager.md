---
name: crew-manager
description: Manages and supervises multiple git-crew tasks. Orchestrates task creation, agent startup, monitoring, review, and merging workflow.
---

# Crew Manager Skill

マネージャーエージェント向けのスキル。複数のタスクを管理・監督し、タスク作成→エージェント起動→監視→レビュー→フィードバック→マージまでのワークフロー全体を実行します。

## 役割

- ワーカーエージェントではなく、**複数タスクを管理・監督するマネージャー**向け
- ユーザーの要望を受けて、タスク作成からマージまでを行う
- タスク実行自体は別のワーカーエージェントに委譲する

---

## ワークフロー

### 1. タスク作成

```bash
# タスク作成
crew new --title "機能Xを実装" --body "詳細な説明..."

# 詳細が不明な場合は調査してから追記
crew edit <id> --body "調査結果を含む詳細計画..."
```

#### タスク作成のベストプラクティス

**タイトル**
- 具体的で簡潔に（例: "crew newの--descを--bodyにリネーム"）
- 動詞から始める（追加、修正、リファクタ等）

**詳細 (--body) に含めるべき内容**
1. **変更対象ファイル**: ファイル名と行番号まで特定できると理想的
2. **実装計画**: ステップごとに分解（1. xxx 2. yyy 3. zzz）
3. **完了条件**: チェックリスト形式で明確に
4. **参考情報**: 関連する既存実装、ドキュメントへのポインタ

**詳細が不明な場合**
- まずタイトルだけでタスク作成
- コードベースを調査して影響範囲を特定
- 調査結果を `crew edit <id> --body "..."` で追記
- 計画が十分に詳細になってから `crew start`

**良い例**
```markdown
## 変更対象ファイル

| ファイル | 箇所 | 変更内容 |
|----------|------|----------|
| internal/cli/task.go | L77, L355 | --desc → --body |
| internal/cli/task_test.go | L66, L534 | テスト更新 |

## 実装計画

1. CLI フラグ定義の変更
2. テストの更新
3. ドキュメントの更新
4. CI 確認

## 完了条件

- [ ] --body フラグが動作する
- [ ] 全テストがパス
- [ ] CI が通る
```

**悪い例**
```
ボタンを追加する
```
→ どこに？何のボタン？どう動く？が不明

---

### 2. タスク起動

```bash
# エージェントでタスク開始
crew start <id> <agent> -m <model>

# 例: opencode + claude-sonnet-4-5
crew start 23 opencode -m anthropic/claude-sonnet-4-5
```

---

### 3. 進捗監視

```bash
# タスク一覧（ステータス確認）
crew list

# セッション出力を確認
crew peek <id>
```

---

### 4. レビュー (in_review になったら)

```bash
# タスク詳細確認
crew show <id>

# 差分確認
crew diff <id>

# worktree内でCI実行
cd <worktree_path> && mise run ci

# 動作確認用にバイナリビルド（必要に応じて）
cd <worktree_path> && go build -o /path/to/binary ./cmd/...
```

**worktreeパスの確認**: `crew show <id>` の出力に含まれる

---

### 5. フィードバック（修正が必要な場合）

```bash
# 1. レビューコメントを追加
crew comment <id> "問題点の説明と修正指示"

# 2. ステータスを戻す
crew edit <id> --status in_progress

# 3. エージェントに指示を送信
crew send <id> "crew show <id> でコメントを確認して修正してください"
crew send <id> Enter   # ← 重要: Enterを別途送信

# 4. 再度 peek で監視
crew peek <id>
```

---

### 6. マージ

```bash
# 確認プロンプトに自動応答
echo "y" | crew merge <id>
```

---

## 重要なポイント

1. **sendの後はEnterを送る**: `crew send <id> "..."` だけでは入力が確定しない
2. **CIはworktree内で実行**: メインリポジトリではなくworktreeに移動してから
3. **in_review→修正の流れ**: comment → edit --status → send → Enter
4. **マージはecho "y"で**: 対話的な確認をスキップ
5. **タスク詳細は具体的に**: 変更対象ファイル・実装計画・完了条件を明記

---

## 利用可能なコマンド

### タスク管理
- `crew list` - タスク一覧
- `crew show <id>` - タスク詳細
- `crew new` - タスク作成
- `crew edit` - タスク編集
- `crew comment` - コメント追加
- `crew close` - タスク終了

### セッション管理
- `crew start` - タスク開始
- `crew stop` - セッション停止
- `crew peek` - セッション出力確認
- `crew send` - キー入力送信
- `crew attach` - セッションにアタッチ

### レビュー・完了
- `crew diff` - 差分確認
- `crew merge` - マージ実行

---

## 制約

- ファイル編集は行わない（read-only mode）
- コードを直接書かない
- ワーカーエージェントに作業を委譲する

---

## ワークフロー例

```bash
# 1. タスク作成
crew new --title "認証機能のリファクタリング" --body "$(cat <<'EOF'
## 変更対象ファイル
- internal/auth/handler.go
- internal/auth/middleware.go

## 実装計画
1. ハンドラーをクリーンアーキテクチャに従って再構成
2. ミドルウェアのエラーハンドリングを改善
3. テストを追加

## 完了条件
- [ ] 全テストがパス
- [ ] CI が通る
EOF
)"

# 2. タスク開始
crew start 1 opencode -m anthropic/claude-sonnet-4-5

# 3. 進捗確認
crew peek 1

# 4. レビュー待ちになったら確認
crew show 1
crew diff 1

# 5. 問題があればフィードバック
crew comment 1 "エラーハンドリングをもう少し詳細に"
crew edit 1 --status in_progress
crew send 1 "crew show 1 でコメントを確認して修正してください"
crew send 1 Enter

# 6. 完了したらマージ
echo "y" | crew merge 1
```
