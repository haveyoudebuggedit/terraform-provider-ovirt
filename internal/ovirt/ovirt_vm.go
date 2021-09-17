package ovirt

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
)

var vmSchema map[string]*schema.Schema = map[string]*schema.Schema{
	"id": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"name": {
		Type:     schema.TypeString,
		Optional: true,
		// TODO implement update
		ForceNew: true,
	},
	"comment": {
		Type:     schema.TypeString,
		Optional: true,
		// TODO implement update
		ForceNew: true,
	},
	"cluster_id": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"template_id": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
}

func (p *provider) vmResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: p.vmCreate,
		ReadContext:   p.vmRead,
		DeleteContext: p.vmDelete,
		Schema:        vmSchema,
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
		params.WithName(name.(string))
	}
	if comment, ok := data.GetOk("comment"); ok {
		params.WithComment(comment.(string))
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
