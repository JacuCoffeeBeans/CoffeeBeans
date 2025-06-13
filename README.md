# CoffeeBeans - コーヒー豆フリマサービス ☕

> 自家焙煎のコーヒー豆を手軽に売買できるフリマサービス

## 概要 (Overview)

自家焙煎のコーヒー豆を手軽に購入・販売できる、CtoC（個人間取引）のフリマサービスです。
高品質なコーヒー豆を焙煎する人と、それを求める人とを繋ぎ、最高のコーヒー体験を提供することを目指します。

## ✨ 主な機能 (Features)

* 高品質な自家焙煎コーヒー豆の売買機能
* 豆の産地、焙煎度合い、精製方法などに基づく詳細な検索・ソート機能など
* 焙煎者の技術や信頼性を評価する独自の評価システム
* 焙煎者向けのアカウント機能（売上・在庫管理など）

## 🛠️ 技術スタック (Tech Stack)

| 分類         | 技術                                |
| :----------- | :---------------------------------- |
| **フロントエンド** | React, TypeScript, Vite, Tailwind CSS |
| **バックエンド** | Go                                  |
| **データベース** | Supabase                            |
| **決済サービス** | Stripe                              |
| **開発環境** | Docker, Dev Containers (VS Code / Cursor) |


## 🚀 開発環境の構築手順 (Installation)

このプロジェクトはDockerとDev Containersを利用して、開発者全員がOSを問わず同じ環境で開発できるように設計されています。

### 1. 事前準備

開発を始める前に、お使いのPCに以下のツールをインストールしてください。

* **Git**: ソースコードを管理します。
* **Docker Desktop**: コンテナを動かすためのエンジンです。
* **VS Code** または **Cursor**: メインのエディタとして使用します。
* **Dev Containers 拡張機能**: VS Code / Cursorの拡張機能マーケットプレイスで `ms-vscode-remote.remote-containers` を検索してインストールします。

#### ⚠️ **Windowsユーザーの方へ**

パフォーマンスと互換性の問題を防ぐため、プロジェクトは必ず**WSL2のファイルシステム内**にクローンしてください。

* **悪い例**: `C:\Users\<ユーザー名>\projects\CoffeeBeans`
* **良い例**: WSLのターミナルを開き、`cd ~` でホームに移動後、`git clone ...` を実行する。（ファイルのパスは `\\wsl.localhost\Ubuntu\home\<ユーザー名>\CoffeeBeans` のようになります）

### 2. セットアップ

1.  **リポジトリをクローン**
    ```bash
    git clone <リポジトリのURL>
    cd CoffeeBeansプロジェクト
    ```

2.  **`.env` ファイルの作成**
    プロジェクトのルートにある `.env.example` ファイルをコピーして、`.env` という名前のファイルを作成します。SupabaseやStripeのAPIキーなど、必要な情報をここに記述します。（現時点では空でも問題ありません）
    ```bash
    cp .env.example .env
    ```

3.  **Dev Containerを起動**
    VS Code (またはCursor) で、クローンした `CoffeeBeansプロジェクト` フォルダを開きます。
    右下に **「Reopen in Container」** という通知が表示されるので、そのボタンをクリックしてください。

    > 初回起動時は、Dockerイメージのビルドに数分かかります。

4.  **依存関係のインストール**
    コンテナの起動後、初回のみ以下のコマンドを実行して、フロントエンドとバックエンドに必要なライブラリをインストールします。

    * **フロントエンド (React)**
        Cursorのターミナル（`ターミナル > 新しいターミナル`）を開き、以下を実行します。このターミナルは`frontend`コンテナに接続されています。
        ```bash
        npm install
        ```

    * **バックエンド (Go)**
        **PCの新しいターミナル (WSL)** を開き、`CoffeeBeansプロジェクト` フォルダにいることを確認してから、以下のコマンドで`backend`コンテナに入り、コマンドを実行します。
        ```bash
        # backendコンテナに入る
        docker compose exec backend sh

        # コンテナ内でGoプロジェクトの依存関係を整理
        go mod tidy

        # コンテナから出る
        exit
        ```

### 3. 開発フロー

セットアップが完了したら、以下の手順で開発サーバーを起動します。

1.  **フロントエンドサーバーの起動**
    Cursorのターミナル（`frontend`コンテナに接続済）で実行します。
    ```bash
    npm run dev
    ```
    → ブラウザで `http://localhost:5173` にアクセスできます。

2.  **バックエンドサーバーの起動**
    **PCの新しいターミナル (WSL)** から`backend`コンテナに入り、`air`を起動します。
    ```bash
    # backendコンテナに入る
    docker compose exec backend sh

    # airを起動（ホットリロードが有効になります）
    air
    ```
    → `http://localhost:8080` でAPIにアクセスできます。

3.  **コーディング**
    あとは、`frontend`および`backend`ディレクトリ内のファイルを編集するだけです。
    ホットリロードが有効になっているため、ファイルを保存すると変更が自動で反映されます。

## 📁 ディレクトリ構成 (Directory Structure)

```
CoffeeBeans/
├── .devcontainer/      # Dev Container（コンテナを使った開発環境）の設定
│   └── devcontainer.json
├── .github/            # GitHub Actions (CI/CD) などの設定
├── backend/            # Goによるバックエンド
│   ├── .air.toml       # ライブリロード設定
│   ├── Dockerfile      # 本番用のコンテナ設定
│   ├── go.mod          # Goの依存パッケージ管理
│   └── main.go         # サーバー起動のエントリポイント
├── docs/               # プロジェクトのドキュメント
├── frontend/           # Reactによるフロントエンド
│   ├── Dockerfile      # 本番用のコンテナ設定
│   ├── package.json    # npmの依存パッケージ・スクリプト管理
│   ├── vite.config.ts  # Vite（ビルドツール）の設定
│   ├── tsconfig.json   # TypeScriptの設定
│   ├── index.html      # エントリーポイントのHTML
│   └── src/            # ソースコードのルート
├── README.md           # あなたが今見ているファイル
└── docker-compose.yml  # 開発環境のコンテナ構成
```
