provider gcore {
  user_name = "test"
  password = "test"


}

resource "gcore_floatingip" "floating_ip" {
  project_id = 1
  region_id = 1
//  fixed_ip_address = "192.168.10.39" // instance`s interface ip
//  port_id = "5c992875-f653-4b7b-af5b-1dc3019e5ffa" //instance`s interface port_id
}


