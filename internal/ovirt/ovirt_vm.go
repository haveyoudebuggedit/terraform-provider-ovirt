package ovirt

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
)

var vmSchema = map[string]*schema.Schema{
	"id": {
		Type:     schema.TypeString,
		Computed: true,
		Description: "oVirt ID of this VM",
	},
	"name": {
		Type:     schema.TypeString,
		Optional: true,
		Description: "User-provided name for the VM. Must only consist of lower- and uppercase letters, numbers, dash, underscore and dot.",
	},
	"comment": {
		Type:     schema.TypeString,
		Optional: true,
		Description: "User-provided comment for the VM.",
	},
	"cluster_id": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
		Description: "Cluster to create this VM on.",
	},
	"template_id": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
		Description: "Base template for this VM.",
	},
}

func (p *provider) vmResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: p.vmCreate,
		ReadContext:   p.vmRead,
		UpdateContext: p.vmUpdate,
		DeleteContext: p.vmDelete,
		Schema:        vmSchema,
		Description:   "The ovirt_vm resource creates a virtual machine in oVirt.",
	}
}

func (p *provider) vmCreate(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	clusterID := data.Get("cluster_id").(string)
	templateID := data.Get("template_id").(string)

	params := ovirtclient.CreateVMParams()
	if name, ok := data.GetOk("name"); ok {
		_, err := params.WithName(name.(string))
		if err != nil {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Invalid VM name: %s", name),
					Detail:   err.Error(),
				},
			}
		}
	}
	if comment, ok := data.GetOk("comment"); ok {
		_, err := params.WithComment(comment.(string))
		if err != nil {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Invalid VM comment: %s", comment),
					Detail:   err.Error(),
				},
			}
		}
	}

	vm, err := p.client.CreateVM(clusterID, templateID, params, ovirtclient.ContextStrategy(ctx))
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to create VM",
				Detail:   err.Error(),
			},
		}
	}

	return vmResourceUpdate(vm, data)
}

func (p *provider) vmRead(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	id := data.Get("id").(string)
	vm, err := p.client.GetVM(id, ovirtclient.ContextStrategy(ctx))
	if err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Failed to fetch VM %s", id),
				Detail:   err.Error(),
			},
		}
	}
	return vmResourceUpdate(vm, data)
}
