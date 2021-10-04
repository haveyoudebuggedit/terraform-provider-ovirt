package ovirt

import (
	"fmt"
	"regexp"
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

func validateNonEmpty(value interface{}, path cty.Path) diag.Diagnostics {
	v, ok := value.(string)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Passed parameter is not a string.",
				Detail:        "The passed parameter is not a string.",
				AttributePath: path,
			},
		}
	}

	if v == "" {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "This field must not be empty.",
				AttributePath: path,
			},
		}
	}
	return nil
}

func validateDiskInterface(i interface{}, path cty.Path) diag.Diagnostics {
	val, ok := i.(string)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "The disk_interface should be a string.",
				Detail:        "The provided disk_interface value is not a string.",
				AttributePath: path,
			},
		}
	}
	interf := ovirtclient.DiskInterface(val)
	if err := interf.Validate(); err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Invalid disk_interface value.",
				Detail:        err.Error(),
				AttributePath: path,
			},
		}
	}
	return nil
}

var uuidRegexp = regexp.MustCompile(`^\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b$`)

func validateUUID(i interface{}, path cty.Path) diag.Diagnostics {
	val, ok := i.(string)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Not a string",
				Detail:        "The specified value is not a string, but must be a string containing a UUID.",
				AttributePath: path,
			},
		}
	}

	if !uuidRegexp.MatchString(val) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Not a UUID",
				Detail:        "The specified value is not a UUID.",
				AttributePath: path,
			},
		}
	}
	return nil
}
