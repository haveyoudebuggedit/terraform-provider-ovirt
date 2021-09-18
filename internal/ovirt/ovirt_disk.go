package ovirt

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var diskSchema = map[string]*schema.Schema{
	"id": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"storagedomain_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "ID of the storage domain to use for disk creation",
		// TODO implement update
		ForceNew: true,
	},
	"format": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "Format for the disk. Must be either 'raw' or 'cow'.",
		// TODO implement update
		ForceNew: true,
	},
	"size": {
		Type:        schema.TypeInt,
		Required:    true,
		Description: "Disk size in bytes.",
		// TODO implement update
		ForceNew: true,
	},
	"alias": {
		Type:     schema.TypeString,
		Optional: true,
		// TODO implement update
		ForceNew: true,
	},
	"sparse": {
		Type:     schema.TypeBool,
		Optional: true,
		// TODO implement update
		ForceNew: true,
	},
	"total_size": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "Size of the actual image size on the disk.",
	},
	"status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Status of the disk. One of 'ok', 'locked', or 'illegal'.",
	},
}
