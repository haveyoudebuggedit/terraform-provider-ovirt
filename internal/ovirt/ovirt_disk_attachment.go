package ovirt

import (
	"context"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
)

var diskAttachmentSchema = map[string]*schema.Schema{
	"id": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"vm_id": {
		Type:             schema.TypeString,
		Required:         true,
		Description:      "ID of the VM the disk should be attached to.",
		ForceNew:         true,
		ValidateDiagFunc: validateUUID,
	},
	"disk_id": {
		Type:             schema.TypeString,
		Required:         true,
		Description:      "ID of the disk to attach. This disk must either be shared or not yet attached to a different VM.",
		ForceNew:         true,
		ValidateDiagFunc: validateUUID,
	},
	"disk_interface": {
		Type:             schema.TypeString,
		Required:         true,
		Description:      "Type of interface to use for attaching disk.",
		ForceNew:         true,
		ValidateDiagFunc: validateDiskInterface,
	},
}

func validateDiskInterface(i interface{}, path cty.Path) diag.Diagnostics {
	val, ok := i.(string)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "The disk_interface should be a string",
				Detail:        "The provided disk_interface value is not a string",
				AttributePath: path,
			},
		}
	}
	interf := ovirtclient.DiskInterface(val)
	if err := interf.Validate(); err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Invalid disk_interface value",
				Detail:        err.Error(),
				AttributePath: path,
			},
		}
	}
	return nil
}

func (p *provider) diskAttachmentResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: p.diskAttachmentCreate,
		ReadContext:   p.diskAttachmentRead,
		DeleteContext: p.diskAttachmentDelete,
		Schema:        diskAttachmentSchema,
		Description:   "The ovirt_disk_attachment resource attaches a single disk to a single VM. For controlling multiple attachments use ovirt_disk_attachments.",
	}
}

func (p *provider) diskAttachmentCreate(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	vmID := data.Get("vm_id").(string)
	diskID := data.Get("disk_id").(string)
	diskInterface := data.Get("disk_interface").(string)

	diskAttachment, err := p.client.CreateDiskAttachment(
		vmID,
		diskID,
		ovirtclient.DiskInterface(diskInterface),
		ovirtclient.CreateDiskAttachmentParams(),
		ovirtclient.ContextStrategy(ctx),
	)
	if err != nil {
		return diag.Diagnostics{diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to create disk attachment",
			Detail:   err.Error(),
		}}
	}

	return diskAttachmentResourceUpdate(diskAttachment, data)
}

func (p *provider) diskAttachmentRead(ctx context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	vmId := data.Get("vm_id").(string)
	attachment, err := p.client.GetDiskAttachment(vmId, data.Id(), ovirtclient.ContextStrategy(ctx))
	if isNotFound(err) {
		data.SetId("")
		return nil
	}
	return diskAttachmentResourceUpdate(attachment, data)
}

func (p *provider) diskAttachmentDelete(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	vmId := data.Get("vm_id").(string)
	if err := p.client.RemoveDiskAttachment(vmId, data.Id(), ovirtclient.ContextStrategy(ctx)); err != nil {
		if isNotFound(err) {
			data.SetId("")
			return nil
		}
		return diag.Diagnostics{diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to remove disk attachment",
			Detail:   err.Error(),
		}}
	}
	data.SetId("")
	return nil
}

func diskAttachmentResourceUpdate(disk ovirtclient.DiskAttachment, data *schema.ResourceData) diag.Diagnostics {
	data.SetId(disk.ID())
	return nil
}