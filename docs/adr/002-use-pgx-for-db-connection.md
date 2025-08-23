# ADR 002: DB 接続に`pgx`を採用

- **ステータス**: 決定 (Accepted)
- **日付**: 2025-08-24

## コンテキスト (背景)

Go バックエンドから Supabase DB (PostgreSQL) への接続にあたり、当初は Supabase 公式が推奨する`supabase-community/postgrest-go`ライブラリを利用していた。
しかし、テスト環境(`go test`)で`Transaction Pooler`経由で接続した際に、`prepared statement already exists`という解決困難なエラーが頻発し、開発が停滞した。

`?prefer_simple_protocol=true`という接続パラメータも試したが、テスト実行のコンテキストでは問題が解決しなかった。

## 決定事項

データベース接続ライブラリを、`postgrest-go`から、より標準的で広く使われている PostgreSQL ドライバである**`jackc/pgx`**に全面的に移行することを決定する。

## 結果 (Consequences)

### メリット

- `pgx`の接続設定で`DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol`を明示的に指定することで、`prepared statement`の問題が完全に解決し、テストとアプリケーションの動作が安定した。
- ライブラリが標準的な SQL クエリを直接書く方式のため、特定のライブラリの独自記法への依存がなくなり、コードの可読性と保守性が向上した。

### デメリット

- `postgrest-go`のような、メソッドチェーン形式のクエリビルダーの利便性は失われる。開発者は SQL を直接記述する必要がある。
