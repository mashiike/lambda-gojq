
resource "aws_iam_role" "gojq" {
  name = "gojq_for_firehose_lambda"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "gojq" {
  role       = aws_iam_role.gojq.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

data "archive_file" "gojq_dummy" {
  type        = "zip"
  output_path = "${path.module}/gojq_dummy.zip"
  source {
    content  = "gojq_dummy"
    filename = "bootstrap"
  }
  depends_on = [
    null_resource.gojq_dummy
  ]
}

resource "null_resource" "gojq_dummy" {}

resource "aws_lambda_function" "gojq_for_firehose" {
  lifecycle {
    ignore_changes = all
  }

  function_name = "gojq_for_firehose"
  role          = aws_iam_role.gojq.arn

  handler  = "bootstrap"
  runtime  = "provided.al2"
  filename = data.archive_file.gojq_dummy.output_path
}

resource "aws_lambda_alias" "current" {
  lifecycle {
    ignore_changes = all
  }
  name             = "current"
  function_name    = aws_lambda_function.gojq_for_firehose.function_name
  function_version = "$LATEST"
}

data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "main" {
  bucket = local.s3_bucket_name
}

resource "aws_s3_bucket_versioning" "main" {
  bucket = aws_s3_bucket.main.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_cloudwatch_log_group" "firehose" {
  name = "/firehose/gojq"

  retention_in_days = 180
}

resource "aws_cloudwatch_log_stream" "default" {
  name           = "default"
  log_group_name = aws_cloudwatch_log_group.firehose.name
}

resource "aws_iam_role" "firehose" {
  name = "gojq_firehose"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "firehose.amazonaws.com"
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = data.aws_caller_identity.current.account_id
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "main" {
  name = "AWSKinesisFirehoseS3Delivery"
  role = aws_iam_role.firehose.id
  policy = jsonencode(
    {
      Version = "2012-10-17",
      Statement = [
        {
          Effect = "Allow",
          Action = [
            "s3:AbortMultipartUpload",
            "s3:GetBucketLocation",
            "s3:GetObject",
            "s3:ListBucket",
            "s3:ListBucketMultipartUploads",
            "s3:PutObject",
          ],
          Resource = [
            aws_s3_bucket.main.arn,
            "${aws_s3_bucket.main.arn}/*",
          ],
        },
        {
          Effect = "Allow",
          Action = [
            "logs:PutLogEvents",
          ],
          Resource = [
            aws_cloudwatch_log_group.firehose.arn,
            "${aws_cloudwatch_log_group.firehose.arn}:*",
          ],
        },
        {
          Effect = "Allow",
          Action = [
            "lambda:InvokeFunction",
            "lambda:GetFunctionConfiguration",
          ],
          Resource = [
            aws_lambda_function.gojq_for_firehose.arn,
            "${aws_lambda_function.gojq_for_firehose.arn}:*",
          ],
        },
      ]
    }
  )
}

resource "aws_kinesis_firehose_delivery_stream" "main" {
  name        = "gojq"
  destination = "extended_s3"

  extended_s3_configuration {
    role_arn   = aws_iam_role.firehose.arn
    bucket_arn = aws_s3_bucket.main.arn

    buffer_size         = 64
    buffer_interval     = 60
    prefix              = "logs/!{timestamp:yyyy}/!{timestamp:MM}/!{timestamp:dd}/!{timestamp:HH}/"
    error_output_prefix = "errors/!{timestamp:yyyy}/!{timestamp:MM}/!{timestamp:dd}/!{timestamp:HH}/!{firehose:error-output-type}/"

    s3_backup_mode = "Enabled"
    s3_backup_configuration {
      role_arn            = aws_iam_role.firehose.arn
      bucket_arn          = aws_s3_bucket.main.arn
      prefix              = "backup/!{timestamp:yyyy}/!{timestamp:MM}/!{timestamp:dd}/!{timestamp:HH}/"
      error_output_prefix = "backup_errors/!{timestamp:yyyy}/!{timestamp:MM}/!{timestamp:dd}/!{timestamp:HH}/!{firehose:error-output-type}/"
    }

    cloudwatch_logging_options {
      enabled         = true
      log_group_name  = aws_cloudwatch_log_stream.default.log_group_name
      log_stream_name = aws_cloudwatch_log_stream.default.name
    }

    processing_configuration {
      enabled = "true"

      processors {
        type = "Lambda"

        parameters {
          parameter_name  = "LambdaArn"
          parameter_value = aws_lambda_alias.current.arn
        }
      }
    }
  }
}
