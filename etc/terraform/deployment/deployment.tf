resource "kubernetes_replication_controller" "deployment" {
  metadata {
    name = "${var.deployment_name}"
    labels {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }
  }

  spec {
    selector {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }

    replicas = "${var.count}"

    template {
      container {
        name  = "${var.deployment_name}"
        image = "${data.aws_ecr_repository.repo.repository_url}:${var.image_tag}"

        resources {
          requests {
            cpu    = "100m"
            memory = "100Mi"
          }
        }

        port {
          protocol       = "TCP"
          container_port = 5000
        }

        env {
          name  = "PORT"
          value = 5000
        }

        env {
          name  = "DEPLOY"
          value = "${var.deployment_name}"
        }

        env {
          name  = "FORWARD_DEST"
          value = "${var.forward_dest}"
        }

        env {
          name  = "FORWARD_COUNT"
          value = "${var.forward_count}"
        }

        env {
          name  = "FORWARD_DEST_CONNECT_TIMEOUT"
          value = "${var.forward_dest_connect_timeout}"
        }

        env {
          name  = "ENFORCE_SSL"
          value = "${var.enforce_ssl}"
        }

        env {
          name  = "LIBRATO_OWNER"
          value = "${var.librato_owner}"
        }

        env {
          name  = "LIBRATO_SOURCE"
          value = "com.herokai.log-iss.${var.deployment_name}"
        }

        env {
          name  = "LIBRATO_TOKEN"
          value = "${var.librato_token}"
        }

        env {
          name  = "TOKEN_MAP"
          value = "${var.token_map}"
        }
      }
    }
  }
}
