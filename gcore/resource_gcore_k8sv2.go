package gcore

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	gcorecloud "github.com/G-Core/gcorelabscloud-go"
	"github.com/G-Core/gcorelabscloud-go/gcore/k8s/v2/clusters"
	"github.com/G-Core/gcorelabscloud-go/gcore/k8s/v2/pools"
	"github.com/G-Core/gcorelabscloud-go/gcore/servergroup/v1/servergroups"
	"github.com/G-Core/gcorelabscloud-go/gcore/task/v1/tasks"
	"github.com/G-Core/gcorelabscloud-go/gcore/volume/v1/volumes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	K8sPoint         = "k8s/clusters"
	tasksPoint       = "tasks"
	K8sCreateTimeout = 3600
)

var k8sCreateTimeout = time.Second * time.Duration(K8sCreateTimeout)

func resourceK8sV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceK8sV2Create,
		ReadContext:   resourceK8sV2Read,
		UpdateContext: resourceK8sV2Update,
		DeleteContext: resourceK8sV2Delete,
		Description:   "Represent k8s cluster with one default pool.",
		Timeouts: &schema.ResourceTimeout{
			Create: &k8sCreateTimeout,
			Update: &k8sCreateTimeout,
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, clusterName, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.Set("name", clusterName)
				d.SetId(clusterName)
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
				DiffSuppressFunc: suppressDiffProjectID,
			},
			"region_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
				DiffSuppressFunc: suppressDiffRegionID,
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Cluster name.",
				Required:    true,
				ForceNew:    true,
			},
			"cni": {
				Type:     schema.TypeList,
				MaxItems: 1,
				MinItems: 1,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  clusters.CalicoProvider.String(),
						},
						"cilium": {
							Type:     schema.TypeList,
							MaxItems: 1,
							MinItems: 1,
							ForceNew: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"mask_size": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"mask_size_v6": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"tunnel": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"encryption": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"lb_mode": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"lb_acceleration": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"routing_mode": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
						}},
				},
			},
			"fixed_network": {
				Type:        schema.TypeString,
				Description: "Fixed network used to allocate network addresses for cluster nodes.",
				Optional:    true,
				ForceNew:    true,
			},
			"fixed_subnet": {
				Type:        schema.TypeString,
				Description: "Fixed subnet used to allocate network addresses for cluster nodes. Subnet should have a router.",
				Optional:    true,
				ForceNew:    true,
			},
			"pods_ip_pool": {
				Type:        schema.TypeString,
				Description: "Pods IPv4 IP pool in CIDR notation.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"services_ip_pool": {
				Type:        schema.TypeString,
				Description: "Services IPv4 IP pool in CIDR notation.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"pods_ipv6_pool": {
				Type:        schema.TypeString,
				Description: "Pods IPv6 IP pool in CIDR notation.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"services_ipv6_pool": {
				Type:        schema.TypeString,
				Description: "Services IPv6 IP pool in CIDR notation.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"keypair": {
				Type:        schema.TypeString,
				Description: "Name of the keypair used for SSH access to nodes.",
				Required:    true,
			},
			"version": {
				Type:        schema.TypeString,
				Description: "Kubernetes version.",
				Required:    true,
			},
			"is_ipv6": {
				Type:        schema.TypeBool,
				Description: "Enable public IPv6 address.",
				Optional:    true,
				ForceNew:    true,
			},
			"pool": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Cluster pool name. Changing the value of this attribute will trigger recreation of the cluster pool.",
							Required:    true,
						},
						"flavor_id": {
							Type:        schema.TypeString,
							Description: "Cluster pool node flavor ID. Changing the value of this attribute will trigger recreation of the cluster pool.",
							Required:    true,
						},
						"min_node_count": {
							Type:        schema.TypeInt,
							Description: "Minimum number of nodes in the cluster pool.",
							Required:    true,
						},
						"servergroup_policy": {
							Type:        schema.TypeString,
							Description: "Server group policy: anti-affinity, soft-anti-affinity or affinity",
							Required:    true,
						},
						"max_node_count": {
							Type:        schema.TypeInt,
							Description: "Maximum number of nodes in the cluster pool.",
							Optional:    true,
							Computed:    true,
						},
						"node_count": {
							Type:        schema.TypeInt,
							Description: "Current node count in the cluster pool.",
							Computed:    true,
						},
						"boot_volume_type": {
							Type:        schema.TypeString,
							Description: "Cluster pool boot volume type. Must be set only for VM pools. Available values are 'standard', 'ssd_hiiops', 'cold', 'ultra'. Changing the value of this attribute will trigger recreation of the cluster pool.",
							Optional:    true,
							Computed:    true,
						},
						"boot_volume_size": {
							Type:        schema.TypeInt,
							Description: "Cluster pool boot volume size. Must be set only for VM pools. Changing the value of this attribute will trigger recreation of the cluster pool.",
							Optional:    true,
							Computed:    true,
						},
						"auto_healing_enabled": {
							Type:        schema.TypeBool,
							Description: "Enable/disable auto healing of cluster pool nodes.",
							Optional:    true,
							Computed:    true,
						},
						"is_public_ipv4": {
							Type:        schema.TypeBool,
							Description: "Assign public IPv4 address to nodes in this pool. Changing the value of this attribute will trigger recreation of the cluster pool.",
							Optional:    true,
							Computed:    true,
						},
						"labels": {
							Type:        schema.TypeMap,
							Description: "Labels applied to the cluster pool nodes.",
							Optional:    true,
							Computed:    true,
						},
						"taints": {
							Type:        schema.TypeMap,
							Description: "Taints applied to the cluster pool nodes.",
							Optional:    true,
							Computed:    true,
						},
						"status": {
							Type:        schema.TypeString,
							Description: "Cluster pool status.",
							Computed:    true,
						},
						"servergroup_name": {
							Type:        schema.TypeString,
							Description: "Server group name",
							Computed:    true,
						},
						"servergroup_id": {
							Type:        schema.TypeString,
							Description: "Server group id",
							Computed:    true,
						},
						"created_at": {
							Type:        schema.TypeString,
							Description: "Cluster pool creation date.",
							Computed:    true,
						},
					},
				},
			},
			"status": {
				Type:        schema.TypeString,
				Description: "Cluster status.",
				Computed:    true,
			},
			"is_public": {
				Type:        schema.TypeBool,
				Description: "True if the cluster is public.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "Cluster creation date.",
				Computed:    true,
			},
			"creator_task_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"task_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceK8sV2Create(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start k8s cluster creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, versionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := clusters.CreateOpts{
		Name:         d.Get("name").(string),
		FixedNetwork: d.Get("fixed_network").(string),
		FixedSubnet:  d.Get("fixed_subnet").(string),
		KeyPair:      d.Get("keypair").(string),
		Version:      d.Get("version").(string),
		IsIPV6:       d.Get("is_ipv6").(bool),
	}

	if cniI, ok := d.GetOk("cni"); ok {
		cniA := cniI.([]interface{})
		cni := cniA[0].(map[string]interface{})
		opts.CNI = &clusters.CNICreateOpts{Provider: clusters.CNIProvider(cni["provider"].(string))}
		if cni["provider"].(string) == "cilium" {
			if ciliumI, ok := cni["cilium"]; ok {
				ciliumA := ciliumI.([]interface{})
				if len(ciliumA) != 0 {
					cilium := ciliumA[0].(map[string]interface{})
					opts.CNI.Cilium = &clusters.CiliumCreateOpts{
						MaskSize:                 cilium["mask_size"].(int),
						MaskSizeV6:               cilium["mask_size_v6"].(int),
						Tunnel:                   clusters.TunnelType(cilium["tunnel"].(string)),
						Encryption:               cilium["encryption"].(bool),
						LoadBalancerMode:         clusters.LBModeType(cilium["lb_mode"].(string)),
						LoadBalancerAcceleration: cilium["lb_acceleration"].(bool),
						RoutingMode:              clusters.RoutingModeType(cilium["routing_mode"].(string)),
					}
				}
			}
		}
	}

	if podsIP, ok := d.GetOk("pods_ip_pool"); ok {
		gccidr, err := parseCIDRFromString(podsIP.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.PodsIPPool = &gccidr
	}

	if svcIP, ok := d.GetOk("services_ip_pool"); ok {
		gccidr, err := parseCIDRFromString(svcIP.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.ServicesIPPool = &gccidr
	}

	if podsIPV6, ok := d.GetOk("pods_ipv6_pool"); ok {
		gccidr, err := parseCIDRFromString(podsIPV6.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.PodsIPV6Pool = &gccidr
	}

	if svcIPV6, ok := d.GetOk("services_ipv6_pool"); ok {
		gccidr, err := parseCIDRFromString(svcIPV6.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		opts.ServicesIPV6Pool = &gccidr
	}

	for _, poolRaw := range d.Get("pool").([]interface{}) {
		pool := poolRaw.(map[string]interface{})
		poolOpts := pools.CreateOpts{
			Name:               pool["name"].(string),
			FlavorID:           pool["flavor_id"].(string),
			MinNodeCount:       pool["min_node_count"].(int),
			MaxNodeCount:       pool["max_node_count"].(int),
			BootVolumeSize:     pool["boot_volume_size"].(int),
			BootVolumeType:     volumes.VolumeType(pool["boot_volume_type"].(string)),
			AutoHealingEnabled: pool["auto_healing_enabled"].(bool),
			IsPublicIPv4:       pool["is_public_ipv4"].(bool),
			ServerGroupPolicy:  servergroups.ServerGroupPolicy(pool["servergroup_policy"].(string)),
		}
		if labels, ok := pool["labels"].(map[string]interface{}); ok {
			poolOpts.Labels = map[string]string{}
			for k, v := range labels {
				poolOpts.Labels[k] = v.(string)
			}
		}
		if taints, ok := pool["taints"].(map[string]interface{}); ok {
			poolOpts.Taints = map[string]string{}
			for k, v := range taints {
				poolOpts.Taints[k] = v.(string)
			}
		}
		opts.Pools = append(opts.Pools, poolOpts)
	}

	results, err := clusters.Create(client, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)

	tasksClient, err := CreateClient(provider, d, tasksPoint, versionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterName, err := tasks.WaitTaskAndReturnResult(tasksClient, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		cluster, err := clusters.Get(client, opts.Name).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot create k8s cluster with name: %s. Error: %w", opts.Name, err)
		}
		return cluster.Name, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterName.(string))
	resourceK8sV2Read(ctx, d, m)

	log.Printf("[DEBUG] Finish k8s cluster creating (%s)", clusterName)
	return diags
}

func resourceK8sV2Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start k8s cluster reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, versionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterName := d.Get("name").(string)
	cluster, err := clusters.Get(client, clusterName).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", cluster.Name)
	d.Set("fixed_network", cluster.FixedNetwork)
	d.Set("fixed_subnet", cluster.FixedSubnet)
	d.Set("keypair", cluster.KeyPair)
	d.Set("version", cluster.Version)
	d.Set("status", cluster.Status)
	d.Set("is_public", cluster.IsPublic)
	d.Set("created_at", cluster.CreatedAt.Format(time.RFC850))
	d.Set("creator_task_id", cluster.CreatorTaskID)
	d.Set("task_id", cluster.TaskID)
	d.Set("is_ipv6", cluster.IsIPV6)
	if cluster.PodsIPPool != nil {
		d.Set("pods_ip_pool", cluster.PodsIPPool.String())
	}
	if cluster.ServicesIPPool != nil {
		d.Set("services_ip_pool", cluster.ServicesIPPool.String())
	}
	if cluster.PodsIPV6Pool != nil {
		d.Set("pods_ipv6_pool", cluster.PodsIPV6Pool.String())
	}
	if cluster.ServicesIPV6Pool != nil {
		d.Set("services_ipv6_pool", cluster.ServicesIPV6Pool.String())
	}

	if cluster.CNI != nil {
		v := map[string]interface{}{
			"provider": cluster.CNI.Provider.String(),
		}
		if cluster.CNI.Cilium != nil {
			v["cilium"] = map[string]interface{}{
				"mask_size":       cluster.CNI.Cilium.MaskSize,
				"mask_size_v6":    cluster.CNI.Cilium.MaskSizeV6,
				"tunnel":          cluster.CNI.Cilium.Tunnel.String(),
				"encryption":      cluster.CNI.Cilium.Encryption,
				"lb_mode":         cluster.CNI.Cilium.LoadBalancerMode.String(),
				"lb_acceleration": cluster.CNI.Cilium.LoadBalancerAcceleration,
				"routing_mode":    cluster.CNI.Cilium.RoutingMode.String(),
			}
		}
		d.Set("cni", []interface{}{v})
	}

	poolMap := map[string]pools.ClusterPool{}
	for _, pool := range cluster.Pools {
		poolMap[pool.Name] = pool
	}

	// Returned pool order needs to match TF state or users will see broken diff,
	// so we first process all pools stored in the state file, and then append any remaining pools.
	var poolData []interface{}
	for _, rawPool := range d.Get("pool").([]interface{}) {
		pool := rawPool.(map[string]interface{})
		poolName := pool["name"].(string)
		if p, ok := poolMap[poolName]; ok {
			poolData = append(poolData, resourceK8sV2PoolDataFromPool(p))
			delete(poolMap, poolName)
		} else {
			// prevent breaking diff when a pool from state file is missing
			log.Printf("[DEBUG] Returning cluster pool placeholder for %q\n", poolName)
			poolData = append(poolData, map[string]interface{}{})
		}
	}
	for _, pool := range poolMap {
		poolData = append(poolData, resourceK8sV2PoolDataFromPool(pool))
	}
	if err := d.Set("pool", poolData); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish k8s cluster reading")
	return diags
}

func resourceK8sV2Update(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start k8s cluster updating")
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, versionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	tasksClient, err := CreateClient(provider, d, tasksPoint, versionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterName := d.Get("name").(string)

	if d.HasChange("version") {
		upgradeOpts := clusters.UpgradeOpts{
			Version: d.Get("version").(string),
		}
		results, err := clusters.Upgrade(client, clusterName, upgradeOpts).Extract()
		if err != nil {
			return diag.FromErr(err)
		}

		taskID := results.Tasks[0]
		log.Printf("[DEBUG] Task id (%s)", taskID)
		_, err = tasks.WaitTaskAndReturnResult(tasksClient, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
			return nil, nil
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("pool") {
		// 1 pool   => Allow in-place updates and add/delete, but return error on replace.
		//             Users must create a new pool with different name in such case.
		// 2+ pools => Allow all operations, but make sure we don't end up with 0 pools at any moment.
		//             This means we process each pool change one-by-one, and perform adds before deletes.
		o, n := d.GetChange("pool")
		old, new := o.([]interface{}), n.([]interface{})

		// Any new pools must be created first, so that "replace" can safely delete pools that it will recreate.
		// This also covers pools that were renamed, because pool name must be unique.
		for _, pool := range new {
			if resourceK8sV2FindClusterPool(old, pool) == nil {
				if err := resourceK8sV2CreateClusterPool(client, tasksClient, clusterName, pool); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		// Check replaces before updates, because replace due to its nature results in all fields being updated.
		for _, pool := range new {
			if resourceK8sV2ClusterPoolNeedsReplace(old, pool) {
				if len(old) == 1 && len(new) == 1 {
					msg := "cannot replace the only pool in the cluster, please create a new pool with different name and remove this one"
					return diag.FromErr(fmt.Errorf("%s", msg))
				}
				if err := resourceK8sV2DeleteClusterPool(client, tasksClient, clusterName, pool); err != nil {
					return diag.FromErr(err)
				}
				if err := resourceK8sV2CreateClusterPool(client, tasksClient, clusterName, pool); err != nil {
					return diag.FromErr(err)
				}
			} else if resourceK8sV2ClusterPoolNeedsUpdate(old, pool) {
				if err := resourceK8sV2UpdateClusterPool(client, clusterName, pool); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		// Finish up by removing all pools that need to be deleted (explicit deletes and leftovers from renames).
		// This allows us to have replace working in case we are going down to 1 pool.
		for _, pool := range old {
			if resourceK8sV2FindClusterPool(new, pool) == nil {
				if err := resourceK8sV2DeleteClusterPool(client, tasksClient, clusterName, pool); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	diags := resourceK8sV2Read(ctx, d, m)
	log.Printf("[DEBUG] Finish k8s cluster updating (%s)", clusterName)
	return diags
}

func resourceK8sV2Delete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start k8s cluster deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, versionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterName := d.Get("name").(string)
	results, err := clusters.Delete(client, clusterName).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)

	tasksClient, err := CreateClient(provider, d, tasksPoint, versionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = tasks.WaitTaskAndReturnResult(tasksClient, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		_, err := clusters.Get(client, clusterName).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete k8s cluster with name: %s", clusterName)
		}
		switch err.(type) {
		case gcorecloud.ErrDefault404:
			return nil, nil
		default:
			return nil, err
		}
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish k8s cluster deleting")
	return diags
}

func resourceK8sV2FindClusterPool(list []interface{}, pool interface{}) interface{} {
	if _, ok := pool.(map[string]interface{}); !ok {
		return nil
	}
	for _, item := range list {
		if _, ok := item.(map[string]interface{}); !ok {
			continue
		}
		if item.(map[string]interface{})["name"] == pool.(map[string]interface{})["name"] {
			return item
		}
	}
	return nil
}

func resourceK8sV2ClusterPoolNeedsUpdate(list []interface{}, pool interface{}) bool {
	found := resourceK8sV2FindClusterPool(list, pool)
	if found == nil {
		return false // adding new pool is not an update
	}
	old, new := found.(map[string]interface{}), pool.(map[string]interface{})
	if old["min_node_count"] != new["min_node_count"] {
		return true
	}
	if old["max_node_count"] != new["max_node_count"] {
		return true
	}
	if old["auto_healing_enabled"] != new["auto_healing_enabled"] {
		return true
	}
	if !reflect.DeepEqual(old["labels"], new["labels"]) {
		return true
	}
	if !reflect.DeepEqual(old["taints"], new["taints"]) {
		return true
	}
	return false
}

func resourceK8sV2ClusterPoolNeedsReplace(list []interface{}, pool interface{}) bool {
	found := resourceK8sV2FindClusterPool(list, pool)
	if found == nil {
		return false // adding new pool is not a replace
	}
	old, new := found.(map[string]interface{}), pool.(map[string]interface{})
	if old["flavor_id"] != new["flavor_id"] {
		return true
	}
	if old["boot_volume_type"] != new["boot_volume_type"] {
		return true
	}
	if old["boot_volume_size"] != new["boot_volume_size"] {
		return true
	}
	if old["is_public_ipv4"] != new["is_public_ipv4"] {
		return true
	}
	if old["servergroup_policy"] != new["servergroup_policy"] {
		return true
	}
	return false
}

func resourceK8sV2CreateClusterPool(client, tasksClient *gcorecloud.ServiceClient, clusterName string, data interface{}) error {
	pool := data.(map[string]interface{})
	poolName := pool["name"].(string)
	log.Printf("[DEBUG] Creating cluster pool (%s)", poolName)

	opts := pools.CreateOpts{
		Name:               pool["name"].(string),
		FlavorID:           pool["flavor_id"].(string),
		MinNodeCount:       pool["min_node_count"].(int),
		MaxNodeCount:       pool["max_node_count"].(int),
		BootVolumeSize:     pool["boot_volume_size"].(int),
		BootVolumeType:     volumes.VolumeType(pool["boot_volume_type"].(string)),
		AutoHealingEnabled: pool["auto_healing_enabled"].(bool),
		ServerGroupPolicy:  servergroups.ServerGroupPolicy(pool["servergroup_policy"].(string)),
		IsPublicIPv4:       pool["is_public_ipv4"].(bool),
	}
	if labels, ok := pool["labels"].(map[string]interface{}); ok {
		opts.Labels = map[string]string{}
		for k, v := range labels {
			opts.Labels[k] = v.(string)
		}
	}
	if taints, ok := pool["taints"].(map[string]interface{}); ok {
		opts.Taints = map[string]string{}
		for k, v := range taints {
			opts.Taints[k] = v.(string)
		}
	}
	results, err := pools.Create(client, clusterName, opts).Extract()
	if err != nil {
		return fmt.Errorf("create cluster pool: %w", err)
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	_, err = tasks.WaitTaskAndReturnResult(tasksClient, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("wait for task %s: %w", taskID, err)
	}

	log.Printf("[DEBUG] Created cluster pool (%s)", poolName)
	return nil
}

func resourceK8sV2DeleteClusterPool(client, tasksClient *gcorecloud.ServiceClient, clusterName string, data interface{}) error {
	pool, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	poolName, ok := pool["name"].(string)
	if !ok || poolName == "" {
		return nil
	}

	log.Printf("[DEBUG] Deleting cluster pool (%s)", poolName)
	results, err := pools.Delete(client, clusterName, poolName).Extract()
	if err != nil {
		switch err.(type) {
		case gcorecloud.ErrDefault404:
			return nil
		default:
			return fmt.Errorf("delete cluster pool: %w", err)
		}
	}

	taskID := results.Tasks[0]
	log.Printf("[DEBUG] Task id (%s)", taskID)
	_, err = tasks.WaitTaskAndReturnResult(tasksClient, taskID, true, K8sCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("wait for task %s: %w", taskID, err)
	}

	log.Printf("[DEBUG] Deleted cluster pool (%s)", poolName)
	return nil
}

func resourceK8sV2UpdateClusterPool(client *gcorecloud.ServiceClient, clusterName string, data interface{}) error {
	pool := data.(map[string]interface{})
	poolName := pool["name"].(string)
	log.Printf("[DEBUG] Updating cluster pool (%s)", poolName)

	opts := pools.UpdateOpts{
		MinNodeCount: pool["min_node_count"].(int),
		MaxNodeCount: pool["max_node_count"].(int),
	}
	if v, ok := pool["auto_healing_enabled"].(bool); ok {
		opts.AutoHealingEnabled = &v
	}
	if labels, ok := pool["labels"].(map[string]interface{}); ok && len(labels) > 0 {
		result := map[string]string{}
		for k, v := range labels {
			result[k] = v.(string)
		}
		opts.Labels = &result
	}
	if taints, ok := pool["taints"].(map[string]interface{}); ok && len(taints) > 0 {
		result := map[string]string{}
		for k, v := range taints {
			result[k] = v.(string)
		}
		opts.Taints = &result
	}
	_, err := pools.Update(client, clusterName, poolName, opts).Extract()
	if err != nil {
		return fmt.Errorf("update cluster pool: %w", err)
	}

	log.Printf("[DEBUG] Updated cluster pool (%s)", poolName)
	return nil
}

func resourceK8sV2PoolDataFromPool(pool pools.ClusterPool) interface{} {
	return map[string]interface{}{
		"name":                 pool.Name,
		"flavor_id":            pool.FlavorID,
		"min_node_count":       pool.MinNodeCount,
		"max_node_count":       pool.MaxNodeCount,
		"node_count":           pool.NodeCount,
		"boot_volume_type":     pool.BootVolumeType.String(),
		"boot_volume_size":     pool.BootVolumeSize,
		"auto_healing_enabled": pool.AutoHealingEnabled,
		"is_public_ipv4":       pool.IsPublicIPv4,
		"labels":               resourceK8sV2FilteredPoolLabels(pool.Labels),
		"taints":               pool.Taints,
		"servergroup_policy":   pool.ServerGroupPolicy,
		"servergroup_name":     pool.ServerGroupName,
		"servergroup_id":       pool.ServerGroupID,
		"status":               pool.Status,
		"created_at":           pool.CreatedAt.Format(time.RFC850),
	}
}

func resourceK8sV2FilteredPoolLabels(labels map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range labels {
		// filter out system labels to hide them from state file and diffs
		if strings.HasPrefix(k, "gcorecluster.x-k8s.io") {
			continue
		}
		result[k] = v
	}
	return result
}
