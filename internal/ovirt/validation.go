package ovirt

import (
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

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

// This function will be used once validating lists is supported in Terraform.
//goland:noinspection GoUnusedFunction
func validateFilesExist(value interface{}, path cty.Path) diag.Diagnostics {
	files, ok := value.([]string)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Passed parameter is not a string.",
				Detail:        "The passed parameter for the file name is not a string.",
				AttributePath: path,
			},
		}
	}

	for _, filename := range files {
		stat, err := os.Stat(filename)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       "File does not exist.",
					Detail:        fmt.Sprintf("The file %s does not exist (%v)", filename, err),
					AttributePath: path,
				},
			}
		}

		if stat.IsDir() {
			return diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       "File is a directory, not a file.",
					Detail:        fmt.Sprintf("Expected %s to be a file, but is a directory.", filename),
					AttributePath: path,
				},
			}
		}
	}

	return nil
}

// This function will be used once validating lists is supported in Terraform.
//goland:noinspection GoUnusedFunction
func validateDirsExist(value interface{}, path cty.Path) diag.Diagnostics {
	dirs, ok := value.([]string)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Passed parameter is not a list of string.",
				Detail:        "The passed parameter for the directories is not a list string.",
				AttributePath: path,
			},
		}
	}

	for _, dirname := range dirs {
		stat, err := os.Stat(dirname)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       "File does not exist.",
					Detail:        fmt.Sprintf("The file %s does not exist (%v)", dirname, err),
					AttributePath: path,
				},
			}
		}

		if !stat.IsDir() {
			return diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       "File is a file, not a directory.",
					Detail:        fmt.Sprintf("Expected %s to be a directory, not a file.", dirname),
					AttributePath: path,
				},
			}
		}
	}

	return nil
}
