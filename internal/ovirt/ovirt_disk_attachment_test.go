package ovirt

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ovirtclient "github.com/ovirt/go-ovirt-client"
	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v2"
)

func TestDiskAttachmentResource(t *testing.T) {
	p := newProvider(ovirtclientlog.NewTestLogger(t))
	storageDomainID := p.testHelper.GetStorageDomainID()
	clusterID := p.testHelper.GetClusterID()
	templateID := p.testHelper.GetBlankTemplateID()

	resource.UnitTest(
		t, resource.TestCase{
			ProviderFactories: p.providerFactories(),
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(
						`
provider "ovirt" {
	mock = true
}

resource "ovirt_disk" "test" {
	storagedomain_id = "%s"
	format           = "raw"
    size             = 512
    alias            = "test"
    sparse           = true
}

resource "ovirt_vm" "test" {
	cluster_id  = "%s"
	template_id = "%s"
}

resource "ovirt_disk_attachment" "test" {
	vm_id          = ovirt_vm.test.id
	disk_id        = ovirt_disk.test.id
	disk_interface = "virtio_scsi"
}
`,
						storageDomainID,
						clusterID,
						templateID,
					),
					Check: resource.ComposeTestCheckFunc(
						resource.TestMatchResourceAttr(
							"ovirt_disk_attachment.test",
							"id",
							regexp.MustCompile("^.+$"),
						),
					),
				},
			},
		},
	)
}

func TestDiskAttachmentResourceImport(t *testing.T) {
	p := newProvider(ovirtclientlog.NewTestLogger(t))
	storageDomainID := p.testHelper.GetStorageDomainID()
	clusterID := p.testHelper.GetClusterID()
	templateID := p.testHelper.GetBlankTemplateID()

	configPart1 := fmt.Sprintf(
		`
provider "ovirt" {
	mock = true
}

resource "ovirt_disk" "test" {
	storagedomain_id = "%s"
	format           = "raw"
    size             = 512
    alias            = "test"
    sparse           = true
}

resource "ovirt_vm" "test" {
	cluster_id  = "%s"
	template_id = "%s"
}
`,
		storageDomainID,
		clusterID,
		templateID,
	)

	configPart2 := fmt.Sprintf(`
%s

resource "ovirt_disk_attachment" "test" {
	vm_id          = ovirt_vm.test.id
	disk_id        = ovirt_disk.test.id
	disk_interface = "virtio_scsi"
}
`, configPart1)

	resource.UnitTest(
		t, resource.TestCase{
			ProviderFactories: p.providerFactories(),
			Steps: []resource.TestStep{
				{
					Config: configPart1,
				},
				{
					Config:       configPart2,
					ImportState:  true,
					ResourceName: "ovirt_disk_attachment.test",
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						diskID := state.RootModule().Resources["ovirt_disk.test"].Primary.Attributes["id"]
						vmID := state.RootModule().Resources["ovirt_vm.test"].Primary.Attributes["id"]

						diskAttachment, err := p.client.CreateDiskAttachment(
							vmID,
							diskID,
							ovirtclient.DiskInterfaceVirtIOSCSI,
							nil,
						)
						if err != nil {
							return "", fmt.Errorf("failed to create test disk attachment (%v)", err)
						}
						return fmt.Sprintf("%s/%s", vmID, diskAttachment.ID()), nil
					},
					Check: resource.ComposeTestCheckFunc(
						resource.TestMatchResourceAttr(
							"ovirt_disk_attachment.test",
							"id",
							regexp.MustCompile("^.+$"),
						),
						resource.TestMatchResourceAttr(
							"ovirt_disk_attachment.test",
							"disk_id",
							regexp.MustCompile("^.+$"),
						),
						resource.TestMatchResourceAttr(
							"ovirt_disk_attachment.test",
							"disk_interface",
							regexp.MustCompile("^.+$"),
						),
						resource.TestMatchResourceAttr(
							"ovirt_disk_attachment.test",
							"vm_id",
							regexp.MustCompile("^.+$"),
						),
					),
				},
			},
		},
	)
}
