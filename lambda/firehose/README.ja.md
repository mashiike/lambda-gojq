## Freihose データ変換 Lambdaのサンプル

このディレクトリは、AWS Lambdaを使ったAmazon Kinesis Firehose データ変換のサンプルです。

### 事前準備

以下のようなconfig.tfを用意しましょう。
```hcl
terraform {
  required_version = "~> 1.4.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 4.67.0"
    }
  }
  backend "s3" {
    bucket = "<your bucket name>"
    key    = "gojq_for_firehose/terraform.tfstate"
    region = "ap-northeast-1"
  }
}

provider "aws" {
  region = "ap-northeast-1"
}

locals {
  s3_bucket_name = "<your bucket name>"
}
```

### 使い方

デプロイするには、以下のコマンドを実行します。

```shell
$ cd lambda/firehose/
$ make terraform/init
$ make terraform/plan
$ make terraform/apply
$ make build
$ make deploy
```

デプロイされるLambda関数の名前は、`gojq_for_firehose` です。

### テスト方法

デプロイされたLambda関数をテストするには、以下のコマンドを実行します。

```shell
$ make put-record
```

すると、 
```json
{"id": "7574922288188224121979983765990975487902977706233495633","timestamp": 1684917788786,"message": "{\"hoge\":\"fuga\"}"}
{"id": "7574922288188224121979983765990975487902977706233495634","timestamp": 1684917788787,"message": "{\"hoge\":\"fuga\"}"} 
```

という内容のレコードが、Firehoseに配信されます。 Lambda関数がFirehoseからInvokeされ変換が実施され、S3には以下のJSONが作成されます。
    
```json
{"hoge":"fuga"}
{"hoge":"fuga"}
```

これは、 gojqの `.message | fromjson` というクエリを各レコードに適用した結果になります。
