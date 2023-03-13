export AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query 'Account' --output text)
.PHONY: clean deploy logs

bootstrap: ../*.go ../go.*
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bootstrap ../*.go

clean:
	rm -f bootstrap
	rm -f *_dummy.zip

deploy: bootstrap
	lambroll --log-level debug deploy --tfstate .terraform/terraform.tfstate

logs:
	lambroll logs --follow --format=short --tfstate .terraform/terraform.tfstate

.PHONY: terraform/init
terraform/init:
	terraform providers lock \
		-platform=darwin_amd64 \
		-platform=darwin_arm64 \
		-platform=linux_amd64 \
		-platform=linux_arm64 \
		-platform=windows_amd64
	terraform init


.PHONY: terraform/plan
terraform/plan:
	terraform plan


.PHONY: terraform/apply
terraform/apply:
	terraform apply

.PHONY: invoke
invoke:
	PAGER="" aws lambda invoke --function-name $(shell tfstate-lookup --state .terraform/terraform.tfstate aws_lambda_function.gojq.arn) --qualifier current --payload file://payload.json  --cli-binary-format raw-in-base64-out output.json
	cat output.json
