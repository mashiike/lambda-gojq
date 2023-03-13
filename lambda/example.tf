
resource "aws_iam_role" "gojq" {
  name = "gojq_lambda"

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

resource "aws_lambda_function" "gojq" {
  lifecycle {
    ignore_changes = all
  }

  function_name = "gojq"
  role          = aws_iam_role.gojq.arn

  handler  = "bootstrap"
  runtime  = "provided.al2"
  filename = data.archive_file.gojq_dummy.output_path
}
