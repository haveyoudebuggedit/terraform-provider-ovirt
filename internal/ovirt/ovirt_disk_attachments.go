package ovirt

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
)

var diskAttachmentsSchema = map[string]*schema.Schema{
	"id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Meta-identifier for the disk attachments. Will always be the same as the VM ID after apply.",
	},
	"attachment": {
		Type:     schema.TypeSet,
		Required: true,
		ForceNew: false,
		Set: func(data interface{}) int {
			return schema.HashString(data.(map[string]interface{})["disk_id"])
		},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"disk_id": {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "ID of the disk to attach. This disk must either be shared or not yet attached to a different VM.",
					ForceNew:         false,
					ValidateDiagFunc: validateUUID,
				},
				"disk_interface": {
					Type:     schema.TypeString,
					Required: true,
					Description: fmt.Sprintf(
						"Type of interface to use for attaching disk. One of: `%s`.",
						strings.Join(ovirtclient.DiskInterfaceValues().Strings(), "`, `"),
					),
					ForceNew:         false,
					ValidateDiagFunc: validateDiskInterface,
				},
			},
		},
	},
	"vm_id": {
		Type:             schema.TypeString,
		Required:         true,
		Description:      "ID of the VM the disks should be attached to.",
		ForceNew:         true,
		ValidateDiagFunc: validateUUID,
	},
	"remove_unmanaged": {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
		Description: `Completely remove attached unmanaged disks, not just detach.

~> Use with care! This option will delete all disks attached to the current VM that are not managed, not just detach them!`,
	},
}

func (p *provider) diskAttachmentsResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: p.diskAttachmentsCreateOrUpdate,
		ReadContext:   p.diskAttachmentsRead,
		UpdateContext: p.diskAttachmentsCreateOrUpdate,
		DeleteContext: p.diskAttachmentsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: p.diskAttachmentsImport,
		},
		Schema:      diskAttachmentsSchema,
		Description: `The ovirt_disk_attachments resource attaches multiple disks to a single VM in one operation. It also allows for removing all attachments that are not declared in an attachment block. This is useful for removing attachments that have been added from the template.

~> Do not use this resource on the same VM as ovirt_disk_attachment (singular). It will cause a ping-pong effect of resources being created and removed on each Terraform run.
`,
	}
}

func (p *provider) diskAttachmentsCreateOrUpdate(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	vmID := data.Get("vm_id").(string)
	desiredAttachments := data.Get("attachment").(*schema.Set)
	removeUnmanaged := data.Get("remove_unmanaged").(bool)
	retry := ovirtclient.ContextStrategy(ctx)

	existingAttachments, err := p.client.ListDiskAttachments(vmID, retry)
	if err != nil {
		return errorToDiags("list existing disk attachments", err)
	}

	diags := diag.Diagnostics{}
	if removeUnmanaged {
		diags = append(
			diags, p.cleanUnmanagedDiskAttachments(
				existingAttachments,
				desiredAttachments,
				removeUnmanaged,
				retry,
			)...,
		)
	}

	for _, desiredAttachmentInterface := range desiredAttachments.List() {
		desiredAttachment := desiredAttachmentInterface.(map[string]interface{})
		diags = append(
			diags, p.createOrUpdateDiskAttachment(
				existingAttachments,
				desiredAttachment,
				vmID,
				retry,
			)...,
		)
	}
	data.SetId(vmID)
	if err := data.Set("attachment", desiredAttachments); err != nil {
		diags = append(diags, errorToDiag("set attachment in Terraform", err))
	}
	return diags
}

func (p *provider) cleanUnmanagedDiskAttachments(
	existingAttachments []ovirtclient.DiskAttachment,
	desiredAttachments *schema.Set,
	removeUnmanaged bool,
	retry ovirtclient.RetryStrategy,
) diag.Diagnostics {
	diags := diag.Diagnostics{}
	for _, existingAttachment := range existingAttachments {
		diags = append(
			diags, p.cleanUnmanagedDiskAttachment(
				desiredAttachments,
				existingAttachment,
				removeUnmanaged,
				retry,
			)...,
		)
	}
	return diags
}

func (p *provider) cleanUnmanagedDiskAttachment(
	desiredAttachments *schema.Set,
	existingAttachment ovirtclient.DiskAttachment,
	removeUnmanaged bool,
	retry ovirtclient.RetryStrategy,
) diag.Diagnostics {
	for _, desiredAttachmentInterface := range desiredAttachments.List() {
		desiredAttachment := desiredAttachmentInterface.(map[string]interface{})
		// We identify by disk ID only since the type will be changed later.
		if desiredAttachment["disk_id"].(string) == existingAttachment.DiskID() {
			return nil
		}
	}
	switch {
	case removeUnmanaged:
		disk, err := existingAttachment.Disk(retry)
		if err != nil {
			return errorToDiags(fmt.Sprintf("get disk for existing disk attachment %s", existingAttachment.ID()), err)
		}
		if err := disk.Remove(retry); err != nil {
			return errorToDiags(fmt.Sprintf("remove disk for disk attachment %s", existingAttachment.ID()), err)
		}
	default:
		if err := existingAttachment.Remove(retry); err != nil {
			return errorToDiags(fmt.Sprintf("remove existing disk attachment %s", existingAttachment.ID()), err)
		}
	}
	return nil
}

func (p *provider) createOrUpdateDiskAttachment(
	existingAttachments []ovirtclient.DiskAttachment,
	desiredAttachment map[string]interface{},
	vmID string,
	retry ovirtclient.RetryStrategy,
) diag.Diagnostics {
	diskID := desiredAttachment["disk_id"].(string)
	diskInterfaceName := desiredAttachment["disk_interface"].(string)

	var foundExisting ovirtclient.DiskAttachment
	for _, existingAttachment := range existingAttachments {
		if existingAttachment.DiskID() == diskID {
			foundExisting = existingAttachment
			break
		}
	}
	if foundExisting != nil {
		if string(foundExisting.DiskInterface()) == diskInterfaceName {
			// Attachment exists and has correct type.
			desiredAttachment["id"] = foundExisting.ID()
			return nil
		}
		// Attachment exists, but has incorrect type. Remove the attachment
		if err := foundExisting.Remove(retry); err != nil {
			return errorToDiags(
				fmt.Sprintf("remove existing disk interface %s", foundExisting.ID()),
				err,
			)
		}
	}
	attachment, err := p.client.CreateDiskAttachment(
		vmID,
		diskID,
		ovirtclient.DiskInterface(diskInterfaceName),
		nil,
		retry,
	)
	if err != nil {
		return errorToDiags(
			fmt.Sprintf("remove existing disk interface %s", foundExisting.ID()),
			err,
		)
	}
	desiredAttachment["id"] = attachment.ID()
	return nil
}

func (p *provider) diskAttachmentsRead(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	vmID := data.Get("vm_id").(string)
	diskAttachments, err := p.client.ListDiskAttachments(vmID, ovirtclient.ContextStrategy(ctx))
	if err != nil {
		return errorToDiags(fmt.Sprintf("listing disk attachments of VM %s", vmID), err)
	}

	attachments := data.Get("attachment").(*schema.Set)
	for _, attachmentInterface := range attachments.List() {
		attachment := attachmentInterface.(map[string]interface{})
		attachments.Remove(attachment)
		found := false
		for _, diskAttachment := range diskAttachments {
			if attachment["disk_id"] == diskAttachment.DiskID() {
				found = true
				attachment["id"] = diskAttachment.ID()
				attachment["disk_interface"] = string(diskAttachment.DiskInterface())
			}
		}
		if found {
			attachments.Add(attachment)
		}
	}
	for _, diskAttachment := range diskAttachments {
		found := false
		for _, attachmentInterface := range attachments.List() {
			attachment := attachmentInterface.(map[string]interface{})
			if attachment["disk_id"] == diskAttachment.DiskID() {
				found = true
				break
			}
		}
		if !found {
			attachments.Add(
				map[string]interface{}{
					"id":             diskAttachment.ID(),
					"disk_id":        diskAttachment.DiskID(),
					"disk_interface": string(diskAttachment.DiskInterface()),
				},
			)
		}
	}
	return nil
}

func (p *provider) diskAttachmentsDelete(
	ctx context.Context,
	data *schema.ResourceData,
	_ interface{},
) diag.Diagnostics {
	diags := diag.Diagnostics{}
	vmID := data.Get("vm_id").(string)
	attachments := data.Get("attachment").(*schema.Set)
	for _, attachmentInterface := range attachments.List() {
		attachment := attachmentInterface.(map[string]interface{})
		if err := p.client.RemoveDiskAttachment(
			vmID,
			attachment["id"].(string),
			ovirtclient.ContextStrategy(ctx),
		); err != nil {
			if !isNotFound(err) {
				diags = append(diags, errorToDiag("remove disk attachment", err))
			} else {
				attachments.Remove(attachment)
			}
		} else {
			attachments.Remove(attachment)
		}
	}
	if err := data.Set("attachment", attachments); err != nil {
		diags = append(diags, errorToDiag("set attachment", err))
	}
	if !diags.HasError() {
		data.SetId("")
	}
	return diags
}

func (p *provider) diskAttachmentsImport(
	ctx context.Context,
	data *schema.ResourceData,
	i interface{},
) ([]*schema.ResourceData, error) {
	id := data.Id()
	if err := data.Set("vm_id", id); err != nil {
		return nil, err
	}
	if diags := p.diskAttachmentsCreateOrUpdate(ctx, data, i); diags.HasError() {
		return nil, diagsToError(diags)
	}
	return []*schema.ResourceData{
		data,
	}, nil
}
