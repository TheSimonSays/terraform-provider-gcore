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

data "gcore_subnet" "tsn" {
  name = "subtest"
  region_id = data.gcore_region.rg.id
  project_id = data.gcore_project.pr.id
}

output "view" {
  value = data.gcore_subnet.tsn
}

