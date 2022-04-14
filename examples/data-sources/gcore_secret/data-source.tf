provider gcore {
  user_name = "test"
  password = "test"


}

data "gcore_project" "pr" {
  name = "test"
}

data "gcore_region" "rg" {
  name = "ED-10 Preprod"
}

data "gcore_secret" "lb_https" {
  name = "lb_https"
  region_id = data.gcore_region.rg.id
  project_id = data.gcore_project.pr.id
}

output "view" {
  value = data.gcore_secret.lb_https
}
