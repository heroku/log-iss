# Placeholder
# -----------
#
# DNS
# ---
# Using https://www.terraform.io/docs/providers/aws/r/route53_record.html
# we could dynamically create a name here, for example (untested):
#
#
# data "aws_route53_zone" "selected" {
#   count = "${var.dns_enabled}" # 0 for false, 1 for true with a default of 0
#
#   name         = "${var.zone}." # e.g. ops.heorkai.com
#   private_zone = false
# }
#
# resource "aws_route53_record" "public" {
#   count = "${var.dns_enabled}" # 0 for false, 1 for true with a default of 0
#
#   zone_id = "${data.aws_route53_zone.selected.zone_id}"
#   name    = "${var.deployment_name}.${data.aws_route53_zone.selected.name}"
#   type    = "CNAME"
#   ttl     = "300"
#   records = [
#     "${kubernetes_service.service_lb.load_balancer_ingress.0.hostname}"
#   ]
# }
