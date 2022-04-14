provider gcore {
  user_name = "test"
  password = "test"
  gcore_platform_api = "https://api.gcorelabs.com"
  gcore_cloud_api = "https://api.gcorelabs.com/cloud"
}

resource "gcore_loadbalancer" "lb" {
  project_id = 1
  region_id = 1
  name = "test"
  flavor = "lb1-1-2"

  listener {
    name = "test3"
    protocol = "HTTP"
    protocol_port = 8080
  }
}

resource "gcore_lblistener" "listener" {
  project_id = 1
  region_id = 1
  name = "test"
  protocol = "TCP"
  protocol_port = 36621
  loadbalancer_id = gcore_loadbalancer.lb.id
}