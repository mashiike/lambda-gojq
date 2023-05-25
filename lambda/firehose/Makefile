export AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query 'Account' --output text)
.PHONY: clean deploy logs

clean:
	rm -f bootstrap
	rm -f *_dummy.zip

deploy: bootstrap
	lambroll --log-level debug deploy --tfstate .terraform/terraform.tfstate

.PHONY:
build:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o bootstrap ../../*.go

export LAMBDA_GOJQ_VERSION=0.2.0
bootstrap:
	curl -sOL https://github.com/mashiike/lambda-gojq/releases/download/v$(LAMBDA_GOJQ_VERSION)/lambda-gojq_$(LAMBDA_GOJQ_VERSION)_linux_arm64.tar.gz && \
	mkdir tmp && \
	tar -zxvf lambda-gojq_$(LAMBDA_GOJQ_VERSION)_linux_arm64.tar.gz -C tmp && \
	mv tmp/bootstrap ./bootstrap && \
	rm -rf tmp && \
	rm -rf lambda-gojq_$(LAMBDA_GOJQ_VERSION)_linux_arm64.tar.gz

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

.PHONY: put-record
put-record:
	PAGER="" aws firehose put-record-batch --delivery-stream-name $(shell tfstate-lookup --state .terraform/terraform.tfstate aws_kinesis_firehose_delivery_stream.main.name) --records file://records.json
