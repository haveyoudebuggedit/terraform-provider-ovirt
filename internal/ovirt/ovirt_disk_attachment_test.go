package ovirt

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v2"
)

func TestDiskAttachmentResource(t *testing.T) {
	p := newProvider(ovirtclientlog.NewTestLogger(t))
	storageDomainID := p.testHelper.GetStorageDomainID()
	clusterID := p.testHelper.GetClusterID()
	templateID := p.testHelper.GetBlankTemplateID()

	resource.UnitTest(t, resource.TestCase{
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
	})
}
