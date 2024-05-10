# 概要

弊社ブログ記事「フロントエンドから Amazon S3 にマルチパートアップロードしたい」のサンプルコードです

# 前提

動作させるためには以下が必要です

### backend

- [Go](https://go.dev/)
- [terraform](https://www.terraform.io/)

### frontend

- [Node.js](https://nodejs.org/en)

# 実行方法

### インフラのセットアップ

`plan`、`apply` の際には以下の指定が要求されますので、ご自身の環境に合わせて指定してください。

- `aws_account_id`: ご自身の AWS アカウント ID
- `developer_provider_name`: 動作させる上では `login.myapp.example.com` など任意の名称で可
- `s3_bucket_name`: 任意の Amazon S3 バケット名

`developer_provider_name` について、実際のアプリケーションでは、認証済みである前提で、アクセスしていきているユーザーを認証した identity provider (IdP)を指し示す名称などを指定します。
例えば、自前で認証処理を行なっている場合、自アプリケーションを示す名称を設定します（[参考](https://docs.aws.amazon.com/cognito/latest/developerguide/developer-authenticated-identities.html)）。

```bash
cd backend/infra # 本 README があるディレクトリから移動する
terraform init
terraform plan
terraform apply

```

以下のように Amazon Cognito の identity pool の id が出力されます。
これらは後続のセットアップで使用します。

```bash
cognito_identity_pool_id = "..."
```

### バックエンドの起動

`.env.sample` をコピーして、`.env` を作成し、各変数を指定します。

```bash
AWS_IDENTITY_POOL_ID=.... # Amazon Congnito identity pool の id (先ほどの手順で output されている値)
AWS_LOGIN_PROVIDER=... # Amazon Cognito identity pool の developer_provider_name (インフラのセットアップで指定した値)
AWS_BUCKET=YOUR_BUCKET_NAME # Amazon S3 のバケット名 (インフラのセットアップで指定した値)
AWS_REGION=ap-northeast-1

```

```bash
cd backend # 本 README があるディレクトリから移動する
go run .

```

### フロントエンドの起動

```bash
cd frontend # 本 README があるディレクトリから移動する
npm install
npm run dev
```
