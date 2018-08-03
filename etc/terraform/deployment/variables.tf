# This controls the naming for all components throughout the cluster.
#

# ----------
# Setup
# ----------
# These is a breaking variable and should only be set once in the tf file
# referencing this configuration or setting up a new environment. If changed,
# it will break an existing setup.
variable "cluster_name" {
  description = "The EKS Cluster Name"
}

variable "deployment_name" {
  description = "Name your log-iss deployment."
  default = "default-log-iss-deployment"
}

variable "image_name" {
  default = "heroku/log-iss"
}

# ----------
# Scale
# ----------
# Changing this adjust the size of the cluster.
variable "count" {
  description = "Number of log-iss instances to run."
  default     = 2
}

# ----------
# Deployment
# ----------
# Changing this deploys a difference image.
variable "image_tag" {
  default = "latest"
}

# ----------
# Configs
# ----------
# These are the base configurations that define the running application. Using
# sensible defaults, unless a customer value is required.

variable "forward_dest" {
  # prompt / error, if not set
  description = "Defines where log-iss should forward events."

  # This should be set dynamically to the syslog loadbalancer ingress hostname.
  #
  # e.g.:
  # forward_dest = "${data.kubernetes_service.syslog.load_balancer_ingress.hostname}:514"
}

variable "forward_count" {
  default = 4
}

variable "forward_dest_connect_timeout" {
  default = 10
}

variable "enforce_ssl" {
  default = true
}

variable "librato_owner" {
  default = "librato-production@heroku.com"
}

# Secrets
# Ideally, these would be auto-populated from vault, aws secrets manager, or
# somewhere else by a plan that's including this as a module.

# !!! DO NOT COMMIT REAL SECRETS TO THIS FILE        !!!
# !!! Replace these values with real values once the !!!
# !!! secret store created.                          !!!
variable "librato_token" {
  default = ""
}

variable "token_map" {
  # prompt / error, if not set
  description = "A pipe delimited string of user/pass values to authenticating to log-iss."
}
