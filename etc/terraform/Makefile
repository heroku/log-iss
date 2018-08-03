example:
	terraform init \
		-backend-config='bucket=$(shell AWS_DEFAULT_REGION=$(AWS_DEFAULT_REGION) \
			AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
			aws s3 ls | grep terraform-backend | awk '{ print $$NF }')'
	terraform apply
