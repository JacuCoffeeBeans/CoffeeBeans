# ベースイメージとしてNode.jsを指定
FROM node:20-bookworm

# 作業ディレクトリを作成
WORKDIR /app

# プロジェクトのソースコードをコピー
# (この時点では空ですが、後で`npm create`で作られます)
COPY . .

# 5173ポートを公開
EXPOSE 5173

# 開発サーバーを起動
# まずはコンテナを起動し続けるためにsleepさせておきます
CMD ["sleep", "infinity"]