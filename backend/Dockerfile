# ベースイメージとしてGoを指定
FROM golang:1.25-bookworm

# 作業ディレクトリを作成
WORKDIR /app

# Goのホットリロードツール「air」をインストール
# (開発効率が上がるため、最初から入れておきましょう)
RUN go install github.com/air-verse/air@latest

# プロジェクトのソースコードをコピー
COPY . .

# 8080ポートを公開
EXPOSE 8080

# こちらもコンテナを起動し続けるためにsleepさせておきます
CMD ["sleep", "infinity"]