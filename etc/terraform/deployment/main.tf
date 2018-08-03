# Typically, this should be used as a module and included as part of an
# environment plan.

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------
# AWS EKS Cluster Data
terraform {
  backend "s3" {
    # When including this as a module, the backend key should be overriden to
    # avoid conflict with other log-iss deployments.
    key = "log-iss-default"
  }
}

provider "aws" {
  version = "~> 1.10"
}

data "aws_eks_cluster" "eks" {
  name = "${var.cluster_name}"
}

data "aws_ecr_repository" "repo" {
  name = "${var.image_name}"
}

# Auth Module
locals {
  # Create locals for shorthand on these, as they're used a few times.
  cert     = "${base64decode(data.aws_eks_cluster.eks.certificate_authority.0.data)}"
  endpoint = "${data.aws_eks_cluster.eks.endpoint}"
}

module "auth" {
  source       = "git@github.com:heroku/splunk-revamp.git//terraform/modules/k8s_auth"
  cluster_name = "${var.cluster_name}"
  endpoint     = "${local.endpoint}"
  cert         = "${local.cert}"
}

# Kubernetes Provider
provider "kubernetes" {
  host                   = "${local.endpoint}"
  cluster_ca_certificate = "${local.cert}"
  token                  = "${module.auth.token}"
  load_config_file       = false
}

# k8s Provider
resource "local_file" "kubeconfig" {
  content  = "${module.auth.kubeconfig}"
  filename = "${path.module}/.${var.cluster_name}.conf"
}

provider "k8s" {
  kubeconfig = "${local_file.kubeconfig.filename}"
}
