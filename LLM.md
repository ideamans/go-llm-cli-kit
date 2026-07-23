# LLM.md — AIエージェント対応標準 v1

アイデアマンズの CLI・ライブラリを「AIエージェントから見つけやすく・使いやすい」
状態にするための社内標準。gridgram と chartjs2img で一巡した実装知見を、
Go 製 CLI 群へ横展開できる形に整理し直したもの。

このリポジトリ（`github.com/ideamans/go-llm-cli-kit`）は、
**この文書（標準）** と **Go 実装のための共有モジュール** と **雛形一式** を同居させている。

- 標準を知りたい → この文書
- Go CLI に実装する → [§5](#5-go-での実装go-llm-cli-kit) と `templates/`
- TypeScript プロジェクトに実装する → [§6](#6-typescript-での実装gridgram--chartjs2img)

---

## 目次

1. [全体像 — 同じ知識を4つの口から配る](#1-全体像--同じ知識を4つの口から配る)
2. [設計原則](#2-設計原則)
3. [`<cli> llm`（CLI から知識を出す）](#3-cli-llmcli-から知識を出す)
4. [`context7.json`](#4-context7json)
5. [Go での実装（go-llm-cli-kit）](#5-go-での実装go-llm-cli-kit)
6. [TypeScript での実装（gridgram / chartjs2img）](#6-typescript-での実装gridgram--chartjs2img)
7. [Agent Skills の配布（Claude plugin / gh skill）](#7-agent-skills-の配布claude-plugin--gh-skill)
8. [プロジェクトローカルの rules / skills](#8-プロジェクトローカルの-rules--skills)
9. [public と private の差分](#9-public-と-private-の差分)
10. [横展開チェックリスト](#10-横展開チェックリスト)
11. [保留事項](#11-保留事項)
12. [裏取りメモ](#12-裏取りメモ)
13. [参考資料](#13-参考資料)

---

## 1. 全体像 — 同じ知識を4つの口から配る

要点は「**1つのナレッジを、4つのフォーマットで配っているだけ**」ということ。

```
                 ┌──────────────────────────────────────────┐
                 │  SSOT: internal/llmdocs/*.md             │
                 │   00-guide.md   手書き（運用知識）        │
                 │   10-*.md       手書き（トピック章）      │
                 │   90-commands.md 生成（コマンドカタログ） │
                 └──────────────────────────────────────────┘
                                    │
        ┌───────────────┬───────────┴────────┬──────────────────┐
        ▼               ▼                    ▼                  ▼
  <cli> llm        context7.json      plugins/<cli>/       同じ SKILL.md を
  （go:embed）      が folders で        skills/*/SKILL.md    gh skill install
                    この dir を指す      → Claude plugin      → Copilot/Cursor/
                                          （marketplace）       Gemini CLI 等
```

| 口 | 到達するエージェント | 前提 |
| --- | --- | --- |
| `<cli> llm` | ローカルで CLI を実行できる全エージェント | バイナリのみ（オフライン可） |
| context7 | context7 MCP を繋いだエージェント | **public リポジトリのみ** |
| Claude plugin | Claude Code | marketplace 登録 |
| `gh skill` | Copilot / Cursor / Gemini CLI / Codex 等 | `gh` v2.90.0+ |

`llms.txt` は現時点では**保留**（[§11](#11-保留事項)）。

---

## 2. 設計原則

### 原則A: SSOT は1か所だけ

仕様の原本はリポジトリ内の**1か所**に置き、他は全てそこから派生させる。
Go CLI では `internal/llmdocs/*.md` がそれにあたる。

| 主題 | 原本 |
| --- | --- |
| 運用知識・落とし穴・ワークフロー | `internal/llmdocs/00-guide.md`（手書き） |
| コマンド・フラグの一覧 | cobra のコマンド定義（→ `90-commands.md` を生成） |
| SKILL.md の本文 | 手書き（生成しない、原則C） |
| context7 の rules | 手書き（生成しない、原則C） |

### 原則B: 派生物は手編集しない

`90-commands.md` を直接編集しても次の `go generate` で消える。コマンド定義を直す。
これを忘れないための仕掛けが [§8](#8-プロジェクトローカルの-rules--skills)。

### 原則C: 生成器は最小本数に絞る

gridgram では当初 SKILL.md と context7 の rules も生成する計画だったが、
実装したら**手書きの方が質が高かった**。理由は、どちらも「読み物」であり
落とし穴の判断が人間の仕事だから。生成するのは**コマンドカタログだけ**でよい。

### 原則D: 派生物を git 追跡するかは早期に決める

- **Go: 追跡する**。`go:embed` はビルド時に実ファイルを要求するため ignore できない。
  代わりに CI で `go generate ./...` → `git diff --exit-code` を必ず置く。
- **TypeScript（gridgram/chartjs2img）: 追跡しない**。ビルド時に毎回生成するため、
  ローカル編集で SSOT がズレた状態のコミットが構造的に起きない。

この2つは正反対だが、どちらも「派生物と SSOT がズレたまま main に乗らない」
という同じ目的を達成している。**中途半端に「追跡するが CI で検査しない」が最悪。**

### 原則E: 配布する SKILL.md には Claude 専用フィールドを書かない

これが gridgram 実装での最大の学び。詳細は [§7-2](#7-2-skillmd-は-agent-skills-標準フィールドのみ)。

---

## 3. `<cli> llm`（CLI から知識を出す）

### 3-1. ねらい

エージェントが CLI を使いこなすのに必要な情報を、**1コマンドで全部**出す。
オフラインで完結し、実行中のバイナリのバージョンと必ず一致する
（ドキュメントサイトと違って絶対に古くならない）。

### 3-2. 確定した仕様

```
<cli> llm                      # Markdown（既定）
<cli> llm --format json        # 章ごとの JSON 配列
<cli> --llm                    # 非推奨。後方互換のため維持（hidden）
```

`--llm` フラグは既存 CLI が持っている形。**サブコマンドへ移行しつつ、
フラグはコマンドラインのどの位置でも動く挙動を維持する**
（`asc apps list --llm` が壊れると既存の利用が止まる）。
`--` 以降は operand として扱い、スキャンしない。

### 3-3. 章立ての規約

ファイル名の数値プレフィックスが章順を決める。

| 範囲 | 用途 | 追跡 |
| --- | --- | --- |
| `00-guide.md` | 冒頭。鉄則・認証・代表ワークフロー・失敗モード | 手書き |
| `10-` 〜 `89-` | トピック章（API スキーマ、ドメイン知識など） | 手書き／必要なら生成 |
| `90-commands.md` | コマンドカタログ | **生成** |

`00-guide.md` に何を書くかは `templates/llmdocs/00-guide.md` を参照。
「エージェントが一番間違えること」を鉄則として先頭に置くのが効く。

---

## 4. `context7.json`

### 4-1. 役割

Upstash の MCP サービス。リポジトリ直下に `context7.json` を置いて
`https://context7.com/add-package` から登録すると、context7 側が md/mdx/txt/rst を
クロールして整形し、MCP 経由で各種エージェントに配信する。

**public リポジトリのみ**が対象。private CLI では使えない（[§9](#9-public-と-private-の差分)）。

### 4-2. 書き方

```json
{
  "$schema": "https://context7.com/schema/context7.json",
  "projectTitle": "<cli>",
  "description": "…",
  "folders": ["internal/llmdocs", "docs"],
  "excludeFolders": ["node_modules", "dist", "vendor", "testdata"],
  "rules": [ "…5〜10本…" ]
}
```

- `folders` に **`internal/llmdocs` を必ず含める**。ここを外すと README しか
  クロールされず、コマンド詳細が context7 検索でヒットしない。
- `rules` は**コードを読めば自明なことは書かない**。文法上・運用上の落とし穴に絞る。
  5〜10本。プロジェクト直下の CLAUDE.md と同質の内容。
- 日本語ドキュメントは除外し、英語を第一言語に据える。

雛形は `templates/context7.json`。

---

## 5. Go での実装（go-llm-cli-kit）

Go CLI 15本で同じコードを書かないための共有モジュール。

```bash
go get github.com/ideamans/go-llm-cli-kit
```

| パッケージ | 役割 |
| --- | --- |
| `llmdocs` | embed した Markdown 章を名前順に結合し、Markdown / JSON で返す |
| `catalog` | cobra のコマンドツリーから Markdown / JSON のコマンドカタログを生成 |
| `llmcmd` | `llm` サブコマンドと、非推奨 `--llm` フラグの後方互換処理 |
| `skillcheck` | 配布用 SKILL.md と plugin.json の検証（`go test` から呼ぶ） |

### 5-1. 配置

```
<cli>/
├── internal/
│   ├── llmdocs/
│   │   ├── llmdocs.go        # //go:embed *.md + //go:generate
│   │   ├── 00-guide.md       # 手書き
│   │   └── 90-commands.md    # 生成物（コミットする）
│   └── gen-llmdocs/
│       └── main.go           # go generate から呼ばれる生成器
├── plugins/<cli>/
│   ├── .claude-plugin/plugin.json
│   ├── skills/<cli>-usage/SKILL.md
│   ├── skills/<cli>-install/SKILL.md
│   └── PUBLISH.md
├── context7.json             # public のみ
└── plugin_test.go            # skillcheck を回す
```

雛形は `templates/` に全てある（`.tmpl` 拡張子はコピー時に外す）。

### 5-2. 組み込み

```go
// internal/llmdocs/llmdocs.go
//go:generate go run ../gen-llmdocs
//go:embed *.md
var files embed.FS

func Docs() *kit.Docs { return kit.New(files, ".") }
```

```go
// main.go
cfg := llmcmd.Config{Docs: llmdocs.Docs()}
llmcmd.AddTo(root, cfg)

// 非推奨 --llm をコマンドラインのどの位置でも受ける
if handled, err := llmcmd.HandleLegacy(os.Args[1:], cfg, os.Stdout); handled {
        if err != nil {
                fmt.Fprintln(os.Stderr, "Error:", err)
                os.Exit(1)
        }
        return
}
```

`HandleLegacy` を `root.Execute()` の**前**に置くこと。cobra に入ってからでは
サブコマンド解決が先に走ってしまう。

### 5-3. 生成器

`cmd.Root()` のように「組み立て済みの cobra ツリーを実行せずに返す」関数が要る。
既存 CLI がパッケージ変数 `rootCmd` を `init()` で組んでいる場合は、それを返す
エクスポート関数を1本足すだけでよい。

```go
md := catalog.Markdown(root, catalog.Options{
        Title: "Command catalog",
        Skip:  []string{"llm"},
})
```

`help` と `completion` は常に除外される。hidden / deprecated なコマンドも既定で除外。
出力は決定的（コマンドとフラグを名前順に整列）なので、差分は実際の変更のみになる。

### 5-4. 検証

```go
report := skillcheck.CheckDir("plugins/<cli>", skillcheck.Options{
        Version:             version,
        Keywords:            []string{"…"},
        RequireInstallSkill: true,
})
```

`Keywords` は「ユーザーがこの道具を必要とするとき口にする語」。
description を書き換えたときにここが落ちるのが正しい挙動で、
**ディスカバリの静かな劣化を検出するための仕掛け**。

### 5-5. CI

`templates/workflows/llm-artifacts.yml` をコピーするか、既存 test ワークフローに以下を追加:

```yaml
- run: go generate ./...
- run: git diff --exit-code    # 派生物が古ければここで落ちる
- run: go test ./...
- run: go run . llm | head -5
```

---

## 6. TypeScript での実装（gridgram / chartjs2img）

Go 版の元になった実装。要点だけ:

- SSOT はテンプレート（`src/templates/llm-reference.template.md`）＋コード＋`examples/`
- 生成器は `scripts/build-*.ts`。`package.json` の `ai:regen` から順に呼ぶ
- 派生物（`src/generated/`）は **gitignore**。CI とバンドル時に毎回作る（原則D）
- `<cli> llm --format markdown|json` は同じ仕様
- SKILL.md 検証は `scripts/validate-plugin-skills.ts`（自前実装）
- `plugin.json.version` は `package.json.version` と一致させ、テストで強制

Go 版との違いは「派生物を追跡するか」だけで、思想は同一。
新規の TS プロジェクトは chartjs2img を雛形にすること。

---

## 7. Agent Skills の配布（Claude plugin / gh skill）

### 7-1. 2リポジトリ構成

```
ideamans/<cli>（本体）
└── plugins/<cli>/              ← プラグイン本体（SKILL.md と plugin.json）

ideamans/claude-public-plugins（または claude-private-plugins）
└── .claude-plugin/marketplace.json   ← 上を git-subdir で参照
```

```json
{
  "name": "<cli>",
  "source": {
    "source": "git-subdir",
    "url": "https://github.com/ideamans/<cli>.git",
    "path": "plugins/<cli>"
  }
}
```

利点は、SKILL.md の更新がリリースと同じ PR で完結し、`git-subdir` が本体の
デフォルトブランチを指すので**マーケットプレイス側を触らずに配信が届く**こと。
裏返しに、本体の破壊的変更は即座に利用者へ波及する。

### 7-2. SKILL.md は Agent Skills 標準フィールドのみ

配布する SKILL.md に Claude 専用フィールドを書かない。標準に絞れば
Claude / Copilot / Cursor / Gemini CLI / Codex のどれでも読める。

| フィールド | 必須 | 備考 |
| --- | --- | --- |
| `name` | Yes | 1–64文字、kebab-case、親ディレクトリ名と一致 |
| `description` | Yes | 1–1024文字。「何をする」と「いつ使う」を両方書く |
| `license` | No | |
| `compatibility` | No | 1–500文字。環境要件（例: `Requires the <cli> binary on PATH`） |
| `metadata` | No | Claude 固有挙動は `metadata.claude-code.*` に逃がす |
| `allowed-tools` | No | 実験的だが各社尊重 |

`argument-hint` / `paths` / `disable-model-invocation` / `model` などは
Claude Code 拡張であり、他エージェントでは無視される。
`skillcheck` がこれを検出して落とす。

**ローカル用**（`.claude/skills/`）は逆に Claude 拡張を自由に使ってよい。
検証対象外。

### 7-3. スキル構成

最低2本。

- `<cli>-usage` — CLI を使って仕事をするワークフロー。本文は**手順**であり
  マニュアルではない（マニュアルは `<cli> llm`）
- `<cli>-install` — GitHub Releases から導入／更新

`-install` は gridgram の運用後に「PATH にいない」事故が読めて追加したもの。
**配布前提のスキルには環境を整えるスキルを必ず1本入れる**。`skillcheck` の
`RequireInstallSkill` で強制する。

`-install` の経路は3本を優先順に固定する。

1. **PATH にある既存バイナリを使う**（最新版の確認はしない。ユーザーが更新を
   求めていない限り API 呼び出しの無駄）。ただし2点だけ確認する:
   コマンド名の衝突（`--version` か `llm` の1行目が当該プロジェクトを名乗るか）と、
   `llm` サブコマンドを持たないほど古くないか
2. **GitHub Releases から取得**。public は `curl`、private は `gh release download`。
   private で `gh` が未認証・権限不足なら**そこで止めてユーザーに `gh auth login` を
   依頼する**。トークンを会話に貼らせる経路は作らない（履歴に残るため）
3. **ソースからビルド**（`go install …@latest`）。Go ツールチェインが要り
   ビルドも走るので最下位。リリースアセットが対象プラットフォームを
   カバーしない場合の逃げ道

インストール先は PATH 上の書き込み可能なディレクトリを優先（`~/.local/bin` →
`/usr/local/bin`）。`sudo` の実行とシェルプロファイルの編集は、エージェントの
判断でやらずユーザーに委ねる。

ドメイン固有の作業が複数あるなら `<cli>-<domain>` を足す（gridgram は4本、
chartjs2img は3本）。

### 7-4. gh skill 兼用

同じ SKILL.md がそのまま使える。

```bash
gh skill install ideamans/<cli>/plugins/<cli>/skills/<cli>-usage --agent claude-code
gh skill install ideamans/<cli>/plugins/<cli>/skills/<cli>-usage --agent copilot
gh skill update
```

`gh skill install` は `repository` / `ref` / tree SHA を frontmatter に注入する
（provenance）。tree SHA で差分検出するため、バージョン bump なしでも
`gh skill update` が更新を検出する。

---

## 8. プロジェクトローカルの rules / skills

本体リポに入った Claude Code セッションを「派生物のズレを作らない共同編集者」に
仕立てる仕掛け。**コミットする**（`.claude/settings.local.json` は除く）。

```
.claude/
├── rules/
│   ├── ai-artifacts-policy.md    # 常駐。派生物の一覧と対応する SSOT の表
│   └── regen-triggers.md         # paths スコープ。SSOT を触った時だけ読まれる
└── skills/
    └── regen-ai/SKILL.md         # /regen-ai：go generate → test → 報告
```

`regen-triggers.md` の frontmatter:

```yaml
paths:
  - "cmd/**/*.go"
  - "internal/llmdocs/0*.md"
  - "internal/llmdocs/1*.md"
```

メッセージは「触ったら `/regen-ai` を実行する。`90-commands.md` を直接いじらない」
だけでよい。常駐ルールにすると認知負荷が上がるので、必要なときだけ出す。

---

## 9. public と private の差分

| 口 | public | private |
| --- | --- | --- |
| `<cli> llm` | ○ | ○ |
| context7 | ○ | **×**（公開クロール前提） |
| Claude plugin | `claude-public-plugins` | `claude-private-plugins` |
| `gh skill install` | ○ | ○（org への read 権限と `gh auth login` が必要） |

private CLI では `context7.json` を置かない。代わりに `00-guide.md` を厚くして
`<cli> llm` 一本で完結させる。install スキルは `curl` ではなく
`gh release download --repo …` を使う（認証が要るため）。

---

## 10. 横展開チェックリスト

1リポジトリ = 1 PR。着手順は依存関係順。

- [ ] `go get github.com/ideamans/go-llm-cli-kit`
- [ ] 既存の `llm.go` / `llmhelp` の本文を `internal/llmdocs/00-guide.md` へ移設
- [ ] `internal/llmdocs/llmdocs.go`（embed + go:generate）を配置
- [ ] `internal/gen-llmdocs/main.go` を配置し、`cmd.Root()` を露出
- [ ] `go generate ./...` → `90-commands.md` をコミット
- [ ] `llmcmd.AddTo` + `llmcmd.HandleLegacy` を main に組み込み、旧 `--llm` の挙動を維持
- [ ] `plugins/<cli>/`（plugin.json + usage/install スキル + PUBLISH.md）
- [ ] `plugin_test.go`（skillcheck）
- [ ] `context7.json`（public のみ）
- [ ] CI に `go generate` → `git diff --exit-code` → `go test` → `go run . llm`
- [ ] README に「AIエージェントから使う」節（plugin install / gh skill / `<cli> llm`）
- [ ] `.claude/rules` と `.claude/skills/regen-ai`
- [ ] marketplace.json に追加（public / private の該当リポ）
- [ ] context7 へ登録（public のみ）

### 受け入れ基準

- `go build ./... && go test ./...` がクリーン
- `go generate ./...` 後に `git diff` が空
- `<cli> llm` と `<cli> llm --format json` が動く
- 旧 `<cli> <subcommand> --llm` が従来どおり動く
- 新規セッションの Claude Code にプラグインだけ渡して、実際に用が足せる（ドッグフード）

### 引っ越し時の落とし穴

- **`cmd.Root()` の露出を忘れる**。`init()` で組み立てる設計だと生成器から
  ツリーを取れない。エクスポート関数を1本足すだけで済む。
- **`--llm` を消してしまう**。既存の利用が壊れる。hidden にして残す。
- **`skillcheck` の `Keywords` を description と一緒に更新しない**。テストが落ちるのが
  正しい挙動なので、落ちたら description 側を見直す。
- **install スキルのアセット名を推測で書く**。goreleaser の設定によって
  OS/arch セグメントの大文字小文字が違う。リリースページを見て確認する。

---

## 11. 保留事項

### llms.txt / llms-full.txt

`llmstxt.org` の公開標準。gridgram と chartjs2img は
`docs/public/llms.txt` として配信済み（VitePress のドキュメントサイトがあるため）。

**Go CLI 群では保留**。ドキュメントサイトを持たないため配信先がなく、
「どこにホストするか」を決めない限り作っても届かないため。
再開する場合の選択肢は、集約ドキュメントサイトを1つ立てる（`/{cli}/llms.txt`）、
各リポで GitHub Pages、リポジトリ直下にコミットして raw 配信、の3つ。

この判断は配信先が決まった時点で見直すこと。それまで Go CLI 側の実装には含めない。

---

## 12. 裏取りメモ

### llms.txt は方向として正しい

Anthropic / Vercel / Next.js / Cloudflare が採用済み。
`llms-full.txt` は Mintlify が Anthropic と共同で策定した派生で、
サイト全 Markdown を1ファイルに連結した版。保留はあくまで配信先の問題。

### context7.json のスキーマ

公式スキーマは `upstash/context7` にある。`folders: []` で全スキャン、
ルート直下の md は常に含まれる、は確定仕様。`rules` の本数上限はスキーマ上ないが、
実務的には5〜10本。

### Agent Skills 標準と Claude Code 拡張

`agentskills.io/specification` の frontmatter は薄い（name / description /
license / compatibility / metadata / allowed-tools）。Claude Code 独自フィールドは
他エージェントでは無視される。配布物は標準のみ、ローカルは自由、の使い分けが要点。

### gh skill

GitHub CLI v2.90.0 から `install / preview / search / publish / update`。
install 時に provenance（repository / ref / tree SHA）が frontmatter に注入され、
tree SHA で差分検出するのでバージョン bump が不要。SSOT 一本管理と相性がよい。

### 着手前と実装後で変わったこと（gridgram）

| 当初計画 | 実装後 |
| --- | --- |
| `gg --llm` フラグ | `gg llm` サブコマンド |
| 生成スクリプト5本 | 3本 |
| context7.json の rules を生成 | 手書き |
| `plugins/` を別リポ | 本体同梱 + marketplace から `git-subdir` 参照 |
| skills 1〜2本 | 4本（`gg-install` を運用後に追加） |
| 外部 `skills-ref` で検証 | 自前実装 |

---

## 13. 参考資料

### 標準・仕様

- llms.txt 仕様: <https://llmstxt.org/>
- Agent Skills オープン標準: <https://agentskills.io/specification>
- Agent Skills 公式バリデータ: <https://github.com/agentskills/agentskills/tree/main/skills-ref>
- context7 スキーマ: <https://github.com/upstash/context7/blob/master/schema/context7.json>
- context7 追加手順: <https://context7.com/docs/adding-libraries>

### Claude Code / gh skill

- Claude Code skills: <https://code.claude.com/docs/en/skills>
- Claude Code plugin marketplace: <https://code.claude.com/docs/ja/plugin-marketplaces>
- Anthropic 公式 skills（marketplace.json 実例）: <https://github.com/anthropics/skills>
- `gh skill` 発表: <https://github.blog/changelog/2026-04-16-manage-agent-skills-with-github-cli/>

### 社内の一次ソース

- `github.com/ideamans/go-llm-cli-kit` — 本標準と Go 実装（このリポジトリ）
- `github.com/ideamans/gridgram` — TS 実装の原型（4スキル、アイコン索引）
- `github.com/ideamans/chartjs2img` — TS 実装（3スキル、llm-docs 分割）
- `github.com/ideamans/claude-public-plugins` / `claude-private-plugins` — marketplace
