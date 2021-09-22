package ovirt

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v2"
)

func TestDiskResource(t *testing.T) {
	p := newProvider(ovirtclientlog.NewTestLogger(t))
	storageDomainID := p.testHelper.GetStorageDomainID()

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: p.providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
provider "ovirt" {
	mock = true
}

resource "ovirt_disk" "foo" {
	storagedomain_id = "%s"
	format           = "raw"
    size             = 512
    alias            = "test"
    sparse           = true
}
`,
					storageDomainID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"ovirt_disk.foo",
						"storagedomain_id",
						regexp.MustCompile(fmt.Sprintf("^%s$", regexp.QuoteMeta(storageDomainID))),
					),
				),
			},
		},
	})
}
