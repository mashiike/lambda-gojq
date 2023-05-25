## Sample of AWS Lambda function for Amazon Kinesis Firehose

This directory is a sample of Amazon Kinesis Firehose data conversion using AWS Lambda.

### Prerequisites

Prepare config.tf as follows.

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

### Usage

To deploy, run the following command.

```shell
$ cd lambda/firehose/
$ make terraform/init
$ make terraform/plan
$ make terraform/apply
$ make build
$ make deploy
```

The name of the deployed Lambda function is `gojq_for_firehose`.

### How to test

To test the deployed Lambda function, run the following command.

```shell
$ make put-record
```

Then,

```json
{"id": "7574922288188224121979983765990975487902977706233495633","timestamp": 1684917788786,"message": "{\"hoge\":\"fuga\"}"}
{"id": "7574922288188224121979983765990975487902977706233495634","timestamp": 1684917788787,"message": "{\"hoge\":\"fuga\"}"} 
```

The following record is delivered to Firehose and invokes the Lambda function.
output S3 object is as follows

```json
{"hoge":"fuga"}
{"hoge":"fuga"}
```

This is the result of applying the query `.message | fromjson` to each record.



