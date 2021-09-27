package ovirt

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	ovirtclient "github.com/ovirt/go-ovirt-client"
)

func validateDiskSize(i interface{}, path cty.Path) diag.Diagnostics {
	size, ok := i.(int)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Disk size must be an integer.",
				Detail:        "The provided disk size is not an integer.",
				AttributePath: path,
			},
		}
	}
	if size <= 0 {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Disk size must be a positive integer.",
				Detail:   fmt.Sprintf("The provided disk size must be a positive integer, got %d.", size),
			},
		}
	}
	return nil
}

func validateFormat(i interface{}, path cty.Path) diag.Diagnostics {
	val, ok := i.(string)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Disk format must be a string.",
				Detail:        "The provided disk format is not a string.",
				AttributePath: path,
			},
		}
	}
	format := ovirtclient.ImageFormat(val)
	if err := format.Validate(); err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Invalid disk image format.",
				Detail:        err.Error(),
				AttributePath: path,
			},
		}
	}
	return nil
}

func validateTLSSystem(value interface{}, path cty.Path) diag.Diagnostics {
	v, ok := value.(bool)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Passed parameter is not a bool.",
				Detail:        "The passed parameter for the system cert pool is not a bool.",
				AttributePath: path,
			},
		}
	}

	if v && runtime.GOOS == "windows" {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "The tls_ca_system option not available on Windows.",
				Detail:        "The tls_ca_system option is not available on Windows due to Golang bug 16736.",
				AttributePath: path,
			},
		}
	}

	return nil
}

func validateTLSInsecure(value interface{}, path cty.Path) diag.Diagnostics {
	v, ok := value.(bool)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Passed parameter is not a bool.",
				Detail:        "The passed parameter for the insecure flag is not a bool.",
				AttributePath: path,
			},
		}
	}

	if v {
		return diag.Diagnostics{
			{
				Severity:      diag.Warning,
				Summary:       "Insecure connection mode enabled.",
				Detail:        "The insecure connection mode to oVirt is enabled. This is a very bad idea because Terraform will not validate the certificate of the oVirt engine.",
				AttributePath: path,
			},
		}
	}
	return nil
}
