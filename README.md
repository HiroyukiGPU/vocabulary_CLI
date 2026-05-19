# vocabulary

ターミナルで使う英単語帳 CLI です。

## Install

このリポジトリを tap してからインストールします。

```sh
brew tap HiroyukiGPU/vocabulary_CLI https://github.com/HiroyukiGPU/vocabulary_CLI
brew install vocabulary
```

`brew install vocabulary` だけで入るのは Homebrew Core に採用された場合だけです。
このリポジトリ単体で配布する場合は、事前に `brew tap` が必要です。

## Usage

```sh
vocabulary
vocabulary add
vocabulary list
vocabulary flip
vocabulary help
```

単語データは `os.UserConfigDir()` 配下の `vocabulary/words.json` に保存されます。
