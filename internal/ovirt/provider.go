package ovirt

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ovirtclient "github.com/ovirt/go-ovirt-client"
	ovirtclientlog "github.com/ovirt/go-ovirt-client-log/v2"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

var providerSchema = map[string]*schema.Schema{
	"username": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Username and realm for oVirt authentication",
	},
	"password": {
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Password for oVirt authentication",
	},
	"url": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "URL for the oVirt engine",
	},
	"extra_headers": {
		Type:        schema.TypeMap,
		Optional:    true,
		Elem:        schema.TypeString,
		Description: "Additional HTTP headers to set on each API call",
	},
	"tls_insecure": {
		Type:             schema.TypeBool,
		Optional:         true,
		ValidateDiagFunc: validateTLSInsecure,
	},
	"tls_system": {
		Type:             schema.TypeBool,
		Optional:         true,
		ValidateDiagFunc: validateTLSSystem,
	},
	"tls_ca_bundle": {
		Type:             schema.TypeString,
		Optional:         true,
		ValidateDiagFunc: validateFilesExist,
	},
	"tls_ca_files": {
		Type:        schema.TypeList,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
	},
	"tls_ca_dirs": {
		Type:        schema.TypeList,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
	},
	"mock": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When set to true, the Terraform provider runs against an internal simulation. This should only be used for testing when an oVirt engine is not available",
		Default:     false,
	},
}

// New returns a new Terraform provider schema for oVirt.
func New() func() *schema.Provider {
	return newProvider(ovirtclientlog.NewNOOPLogger()).provider
}

func newProvider(logger ovirtclientlog.Logger) *provider {
	helper, err := ovirtclient.NewTestHelper(
		"https://localhost/ovirt-engine/api",
		"admin@internal",
		"",
		ovirtclient.TLS().Insecure(),
		"",
		"",
		"",
		"",
		true,
		logger,
	)
	if err != nil {
		panic(err)
	}
	return &provider{
		testHelper: helper,
	}
}

type provider struct {
	testHelper ovirtclient.TestHelper
	client     ovirtclient.Client
}

func (p *provider) provider() *schema.Provider {
	return &schema.Provider{
		Schema:               providerSchema,
		ConfigureContextFunc: p.configureProvider,
		ResourcesMap: map[string]*schema.Resource{
			"ovirt_vm": p.vmResource(),
		},
		DataSourcesMap: map[string]*schema.Resource{},
	}
}

func (p *provider) providerFactories() map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		"ovirt": func() (*schema.Provider, error) {
			return p.provider(), nil
		},
	}
}

func (p *provider) configureProvider(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if mock, ok := data.GetOk("mock"); ok && mock == true {
		p.client = p.testHelper.GetClient()
		return p, diags
	}

	url, diags := extractString(data, "url", diags)
	username, diags := extractString(data, "username", diags)
	password, diags := extractString(data, "password", diags)

	tls := ovirtclient.TLS()
	if insecure, ok := data.GetOk("tls_insecure"); ok && insecure == true {
		tls.Insecure()
	}
	if system, ok := data.GetOk("tls_system"); ok && system == true {
		tls.CACertsFromSystem()
	}
	if caFiles, ok := data.GetOk("tls_ca_files"); ok {
		caFileList, ok := caFiles.([]string)
		if !ok {
			diags = append(
				diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "The tls_ca_files option is not a list of files",
					Detail:   "The tls_ca_files option must be a list of files containing PEM-formatted certificates",
				},
			)
		} else {
			for _, caFile := range caFileList {
				tls.CACertsFromFile(caFile)
			}
		}
	}
	if caDirs, ok := data.GetOk("tls_ca_dirs"); ok {
		caDirList, ok := caDirs.([]string)
		if !ok {
			diags = append(
				diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "The tls_ca_dirs option is not a list of files",
					Detail:   "The tls_ca_dirs option must be a list of files containing PEM-formatted certificates",
				},
			)
		} else {
			for _, caDir := range caDirList {
				tls.CACertsFromDir(caDir)
			}
		}
	}
	if caBundle, ok := data.GetOk("tls_ca_bundle"); ok {
		caCerts, ok := caBundle.(string)
		if !ok {
			diags = append(
				diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "The tls_ca_bundle option is not a string",
					Detail:   "The tls_ca_bundle option must be a string containing the CA certificates in PEM format",
				},
			)
		} else {
			tls.CACertsFromMemory([]byte(caCerts))
		}
	}

	if len(diags) != 0 {
		return nil, diags
	}

	client, err := ovirtclient.New(
		url,
		username,
		password,
		tls,
		ovirtclientlog.NewNOOPLogger(),
		nil,
	)
	if err != nil {
		diags = append(
			diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "Failed to create oVirt client",
				Detail:        err.Error(),
				AttributePath: nil,
			},
		)
		return nil, diags
	}
	p.client = client
	return p, diags
}

// vmResourceUpdate takes the VM object and converts it into Terraform resource data.
func vmResourceUpdate(vm ovirtclient.VM, data *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	data.SetId(vm.ID())
	if err := data.Set("cluster_id", vm.ClusterID()); err != nil {
		diags = append(
			diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to update cluster_id field",
				Detail:   err.Error(),
			},
		)
	}
	if err := data.Set("template_id", vm.TemplateID()); err != nil {
		diags = append(
			diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to update template_id field",
				Detail:   err.Error(),
			},
		)
	}
	if err := data.Set("name", vm.Name()); err != nil {
		diags = append(
			diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to update name field",
				Detail:   err.Error(),
			},
		)
	}
	if err := data.Set("comment", vm.Comment()); err != nil {
		diags = append(
			diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Failed to update comment field",
				Detail:   err.Error(),
			},
		)
	}
	return diags
}

func (p *provider) vmDelete(ctx context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	if err := p.client.RemoveVM(data.Id(), ovirtclient.ContextStrategy(ctx)); err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("Failed to remove VM %s", data.Id()),
				Detail:        err.Error(),
				AttributePath: nil,
			},
		}
	}
	return nil
}

func extractString(data *schema.ResourceData, option string, diags diag.Diagnostics) (string, diag.Diagnostics) {
	var url string
	urlInterface, ok := data.GetOk("url")
	if !ok {
		diags = append(
			diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("The %s option is not set", option),
				Detail:   fmt.Sprintf("The %s option must be set if mock=false", option),
			},
		)
	} else {
		url, ok = urlInterface.(string)
		if !ok {
			diags = append(
				diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("The %s option is not a string", option),
					Detail:   fmt.Sprintf("The %s option must be set and be a string", option),
				},
			)
		}
	}
	return url, diags
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

	if v == true && runtime.GOOS == "windows" {
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
