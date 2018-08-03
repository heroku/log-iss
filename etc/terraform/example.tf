# This is an example "default" deployment of deployment/*.tf
#
# Run with:
# $ terraform init -backend-config='bucket=<bucket name>'
# $ terraform apply
terraform {
  backend "s3" {
    # When including this as a module, the backend key should be overriden to
    # avoid conflict with other log-iss deployments.
    key = "log-iss-example"
  }
}

module "deployment" {
  source = "deployment"

  cluster_name    = "eks"
  deployment_name = "log-iss-example"
  forward_dest    = "localhost:514" # doesn't work

  # !!! DO NOT COMMIT REAL SECRETS TO THIS FILE        !!!
  # !!! Replace these values with real values once the !!!
  # !!! secret store created.                          !!!
  token_map = "ingress:changeme"
}
