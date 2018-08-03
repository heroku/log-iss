resource "kubernetes_service" "service_lb" {
  metadata {
    name = "${var.deployment_name}-lb"
    labels {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }
  }

  spec {
    type = "LoadBalancer"

    selector {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }

    port {
      name        = "tcp"
      protocol    = "TCP"
      port        = 5000
      target_port = 5000
    }
  }
}

output "lb_hostname" {
  value = "${kubernetes_service.service_lb.load_balancer_ingress.0.hostname}"
}

resource "kubernetes_service" "service" {
  metadata {
    name = "${var.deployment_name}"
    labels {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }
  }

  spec {
    cluster_ip = "None"

    selector {
      app  = "${var.deployment_name}-app"
      role = "${var.deployment_name}-role"
    }

    port {
      name        = "tcp"
      protocol    = "TCP"
      port        = 5000
      target_port = 5000
    }
  }
}
