package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/ovirt/terraform-provider-ovirt-ng/internal/ovirt"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ovirt.New(),
	})
}
