package ovirt

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v2"
)

func TestVMResource(t *testing.T) {
	p := newProvider(ovirtclientlog.NewTestLogger(t))
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
					
					resource "ovirt_vm" "foo" {
						cluster_id = "%s"
						template_id = "%s"
					}`,
					clusterID,
					templateID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"ovirt_vm.foo",
						"cluster_id",
						regexp.MustCompile(fmt.Sprintf("^%s$", regexp.QuoteMeta(clusterID))),
					),
					resource.TestMatchResourceAttr(
						"ovirt_vm.foo",
						"template_id",
						regexp.MustCompile(fmt.Sprintf("^%s$", regexp.QuoteMeta(templateID))),
					),
				),
			},
		},
	})
}

type testVM struct {
	id string
	name string
	comment string
	clusterID string
	templateID string

}

func (t *testVM) ID() string {
	return t.id
}

func (t *testVM) Name() string {
	return t.name
}

func (t *testVM) Comment() string {
	return t.comment
}

func (t *testVM) ClusterID() string {
	return t.clusterID
}

func (t *testVM) TemplateID() string {
	return t.templateID
}

func (t *testVM) Status() ovirtclient.VMStatus {
	panic("implement me")
}

func (t *testVM) Remove(retries ...ovirtclient.RetryStrategy) error {
	panic("not implemented")
}

func (t *testVM) CreateNIC(name string, vnicProfileID string, retries ...ovirtclient.RetryStrategy) (
	ovirtclient.NIC,
	error,
) {
	panic("not implemented")
}

func (t *testVM) GetNIC(id string, retries ...ovirtclient.RetryStrategy) (ovirtclient.NIC, error) {
	panic("not implemented")
}

func (t *testVM) ListNICs(retries ...ovirtclient.RetryStrategy) ([]ovirtclient.NIC, error) {
	panic("not implemented")
}

func (t *testVM) AttachDisk(
	diskID string,
	diskInterface ovirtclient.DiskInterface,
	params ovirtclient.CreateDiskAttachmentOptionalParams,
	retries ...ovirtclient.RetryStrategy,
) (ovirtclient.DiskAttachment, error) {
	panic("not implemented")
}

func (t *testVM) GetDiskAttachment(
	diskAttachmentID string,
	retries ...ovirtclient.RetryStrategy,
) (ovirtclient.DiskAttachment, error) {
	panic("not implemented")
}

func (t *testVM) ListDiskAttachments(retries ...ovirtclient.RetryStrategy) ([]ovirtclient.DiskAttachment, error) {
	panic("not implemented")
}

func (t *testVM) DetachDisk(diskAttachmentID string, retries ...ovirtclient.RetryStrategy) error {
	panic("not implemented")
}

func TestVMResourceUpdate(t *testing.T) {
	vm := &testVM{
		id: "asdf",
		name: "test VM",
		comment: "This is a test VM.",
		clusterID: "cluster-1",
		templateID: "template-1",
	}
	resourceData := schema.TestResourceDataRaw(t, vmSchema, map[string]interface{}{})
	diags := vmResourceUpdate(vm, resourceData)
	if len(diags) != 0 {
		t.Fatalf("failed to convert VM resource (%v)", diags)
	}
	compareResource(t, resourceData, "id", vm.id)
	compareResource(t, resourceData, "name", vm.name)
	compareResource(t, resourceData, "cluster_id", vm.clusterID)
	compareResource(t, resourceData, "template_id", vm.templateID)
}

func compareResource(t *testing.T, data *schema.ResourceData, field string, value string) {
	if resourceValue := data.Get(field); resourceValue != value {
		t.Fatalf("invalid resource %s: %s, expected: %s", field, resourceValue, value)
	}
}