package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/haveyoudebuggedit/terraform-provider-ovirt/ovirt"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: ovirt.New(),
	})
}
