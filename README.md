# lambda-gojq

![Latest GitHub release](https://img.shields.io/github/release/mashiike/lambda-gojq.svg)
![Github Actions test](https://github.com/mashiike/lambda-gojq/workflows/Test/badge.svg?branch=main)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/lambda-gojq/blob/master/LICENSE)

AWS Lambda bootstrap for https://github.com/itchyny/gojq

## Usage with AWS Lambda (serverless)

Let's solidify the Lambda package with the following zip arcive (runtime `provided.al2`)

```
lambda.zip
└── bootstrap  
```

A related document is [https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-custom.html)

for example.

deploy lambda functions, in [lambda directory(default mode)](lambda/default)  
The example of lambda directory uses [lambroll](https://github.com/fujiwara/lambroll) for deployment.

For more information on the infrastructure around lambda functions, please refer to [example.tf](lambda/default/example.tf).

```shell
$ cd lambda/
$ make terraform/init
$ make terraform/plan
$ make terraform/apply
$ make deploy
```

## lambda Payload (MODE=default)

for example
```json
{
  "query": ". | .time=(now | strftime(\"%Y-%m-%dT%%H:%M:%SZ\"))",
  "data": {
    "env": "pord",
    "port": 80
  }
}
```

output 
```json
{"env":"pord","hoge":"2023-03-13T%H:32:52Z","port":80}    
```

## LICENSE

MIT 
