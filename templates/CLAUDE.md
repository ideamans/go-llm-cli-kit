# CLAUDE.md — <cli>

<!--
TEMPLATE. Copy to CLAUDE.md at the root of the CLI repository and replace every
<placeholder>. If the repository already has a CLAUDE.md, append the
"変更時の必須手順" section to it rather than overwriting.
-->

<何をする CLI か 1〜2 文。バイナリ名がリポジトリ名と違う場合はここで明記する。>

## 変更時の必須手順

**機能を追加した、フラグを増やした、既存の挙動を変えた — このいずれかをしたら、
3か所すべてを更新してから終わること。** どれか1つでも欠けると、人間か
エージェントのどちらかが古い情報で動くことになる。

| 更新先 | 対象 | やり方 |
| --- | --- | --- |
| ① ドキュメント | `README.md` / `README_ja.md` | 手で更新。使い方が変わったときのみ |
| ② ヘルプ | cobra の `Short` / `Long` / フラグ説明 | コード内。エージェントも人間もまずここを読む |
| ③ **LLMナレッジ** | `internal/llmdocs/00-guide.md` | 手書き。鉄則・認証・ワークフロー・失敗モードが変わったら |
| | `internal/llmdocs/9*-*.md` | **生成物。手編集しない** → `go generate ./...` |
| | `plugins/<cli>/skills/*/SKILL.md` | 手順や前提が変わったとき |
| | `context7.json` の `rules` | 新しい落とし穴が生まれたとき |

③ を忘れやすい。ドキュメントとヘルプは人間が読んで気づくが、**LLMナレッジが
古いことには誰も気づかない**（エージェントが黙って間違えるだけ）。

判断に迷ったときの目安:

- 新しいサブコマンド／フラグを足した → ② と `go generate`。使い方が非自明なら ③ の
  `00-guide.md` にも
- 既存の挙動を変えた（既定値、出力形式、終了コード） → ①②③ すべて。特に
  `00-guide.md` の「出力契約」と `context7.json` の該当 rule
- エージェントが間違えやすい罠を見つけた → `00-guide.md` の失敗モード表と
  `context7.json` の `rules`
- 破壊的操作を追加した → `SKILL.md` に確認手順を書く。CLI 側に確認プロンプトが
  無いなら、そのことを明記する

## 確認

```bash
go generate ./...     # 生成物を作り直す
git diff --exit-code  # 差分が出たらコミット漏れ
go test ./...         # SKILL.md 検証とバージョン整合を含む
go run . llm | head   # 埋め込みリファレンスが壊れていないか
```

CI も同じことをする。ローカルで通してから push すること。

## 参照

- 標準: <https://github.com/ideamans/go-llm-cli-kit/blob/main/LLM.md>
- 生成物と原本の対応: `.claude/rules/ai-artifacts-policy.md`
- 再生成: `/regen-ai`
