package azure

import (
	"testing"

	"github.com/drone-runners/drone-runner-aws/app/oshelp"
	"github.com/drone-runners/drone-runner-aws/types"
)

func TestWithPrivateIP(t *testing.T) {
	tests := []struct {
		name      string
		privateIP bool
		want      bool
	}{
		{
			name:      "enable private IP",
			privateIP: true,
			want:      true,
		},
		{
			name:      "disable private IP",
			privateIP: false,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithPrivateIP(tt.privateIP)
			opt(c)
			if c.privateIP != tt.want {
				t.Errorf("WithPrivateIP() = %v, want %v", c.privateIP, tt.want)
			}
		})
	}
}

func TestWithVNet(t *testing.T) {
	tests := []struct {
		name     string
		vnetName string
		want     string
	}{
		{
			name:     "set vnet name",
			vnetName: "my-existing-vnet",
			want:     "my-existing-vnet",
		},
		{
			name:     "empty vnet name",
			vnetName: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithVNet(tt.vnetName)
			opt(c)
			if c.vnetName != tt.want {
				t.Errorf("WithVNet() = %v, want %v", c.vnetName, tt.want)
			}
		})
	}
}

func TestWithSubnet(t *testing.T) {
	tests := []struct {
		name       string
		subnetName string
		want       string
	}{
		{
			name:       "set subnet name",
			subnetName: "my-existing-subnet",
			want:       "my-existing-subnet",
		},
		{
			name:       "empty subnet name",
			subnetName: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithSubnet(tt.subnetName)
			opt(c)
			if c.subnetName != tt.want {
				t.Errorf("WithSubnet() = %v, want %v", c.subnetName, tt.want)
			}
		})
	}
}

func TestUseExistingNetwork(t *testing.T) {
	tests := []struct {
		name             string
		vnetName         string
		subnetName       string
		existingSubnetID string
		wantUseExisting  bool
	}{
		{
			name:            "both vnet and subnet provided",
			vnetName:        "my-vnet",
			subnetName:      "my-subnet",
			wantUseExisting: true,
		},
		{
			name:             "subnet id only",
			existingSubnetID: "/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s",
			wantUseExisting:  true,
		},
		{
			name:            "only vnet provided",
			vnetName:        "my-vnet",
			subnetName:      "",
			wantUseExisting: false,
		},
		{
			name:            "only subnet provided",
			vnetName:        "",
			subnetName:      "my-subnet",
			wantUseExisting: false,
		},
		{
			name:            "neither provided",
			vnetName:        "",
			subnetName:      "",
			wantUseExisting: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				vnetName:         tt.vnetName,
				subnetName:       tt.subnetName,
				existingSubnetID: tt.existingSubnetID,
			}
			if c.useExistingNetwork() != tt.wantUseExisting {
				t.Errorf("useExistingNetwork = %v, want %v", c.useExistingNetwork(), tt.wantUseExisting)
			}
		})
	}
}

func TestSetPlatformDefaults(t *testing.T) {
	tests := []struct {
		name     string
		platform *types.Platform
		want     *types.Platform
		wantErr  bool
	}{
		{
			name:     "empty platform defaults to Windows amd64",
			platform: &types.Platform{},
			want: &types.Platform{
				Arch: oshelp.ArchAMD64,
				OS:   oshelp.OSWindows,
			},
			wantErr: false,
		},
		{
			name: "valid Windows platform",
			platform: &types.Platform{
				Arch: oshelp.ArchAMD64,
				OS:   oshelp.OSWindows,
			},
			want: &types.Platform{
				Arch: oshelp.ArchAMD64,
				OS:   oshelp.OSWindows,
			},
			wantErr: false,
		},
		{
			name: "valid ARM64 platform",
			platform: &types.Platform{
				Arch: oshelp.ArchARM64,
				OS:   oshelp.OSWindows,
			},
			want: &types.Platform{
				Arch: oshelp.ArchARM64,
				OS:   oshelp.OSWindows,
			},
			wantErr: false,
		},
		{
			name: "invalid arch",
			platform: &types.Platform{
				Arch: "invalid",
				OS:   oshelp.OSWindows,
			},
			want: &types.Platform{
				Arch: "invalid",
				OS:   oshelp.OSWindows,
			},
			wantErr: true,
		},
		{
			name: "invalid OS (Linux not supported)",
			platform: &types.Platform{
				Arch: oshelp.ArchAMD64,
				OS:   oshelp.OSLinux,
			},
			want: &types.Platform{
				Arch: oshelp.ArchAMD64,
				OS:   oshelp.OSLinux,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetPlatformDefaults(tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPlatformDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Arch != tt.want.Arch {
				t.Errorf("SetPlatformDefaults() Arch = %v, want %v", got.Arch, tt.want.Arch)
			}
			if got.OS != tt.want.OS {
				t.Errorf("SetPlatformDefaults() OS = %v, want %v", got.OS, tt.want.OS)
			}
		})
	}
}

func TestDriverName(t *testing.T) {
	c := &config{}
	got := c.DriverName()
	want := string(types.Azure)
	if got != want {
		t.Errorf("DriverName() = %v, want %v", got, want)
	}
}

func TestCanHibernate(t *testing.T) {
	c := &config{}
	got := c.CanHibernate()
	want := false
	if got != want {
		t.Errorf("CanHibernate() = %v, want %v", got, want)
	}
}

func TestInstanceType(t *testing.T) {
	tests := []struct {
		name  string
		offer string
		want  string
	}{
		{
			name:  "Windows Server offer",
			offer: "WindowsServer",
			want:  "WindowsServer",
		},
		{
			name:  "empty offer",
			offer: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{offer: tt.offer}
			got := c.InstanceType()
			if got != tt.want {
				t.Errorf("InstanceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRootDir(t *testing.T) {
	tests := []struct {
		name    string
		rootDir string
		want    string
	}{
		{
			name:    "Windows temp dir",
			rootDir: "C:\\Windows\\Temp\\azure",
			want:    "C:\\Windows\\Temp\\azure",
		},
		{
			name:    "empty root dir",
			rootDir: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{rootDir: tt.rootDir}
			got := c.RootDir()
			if got != tt.want {
				t.Errorf("RootDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZones(t *testing.T) {
	tests := []struct {
		name  string
		zones []*string
		want  string
	}{
		{
			name:  "single zone",
			zones: []*string{strPtr("1")},
			want:  "1,", // Note: existing implementation adds trailing comma
		},
		{
			name:  "multiple zones",
			zones: []*string{strPtr("1"), strPtr("2")},
			want:  "1,2,",
		},
		{
			name:  "no zones",
			zones: []*string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{zones: tt.zones}
			got := c.Zones()
			if got != tt.want {
				t.Errorf("Zones() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
	}{
		{
			name:     "custom location",
			location: "westus2",
			want:     "westus2",
		},
		{
			name:     "empty location defaults to eastus2",
			location: "",
			want:     "eastus2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithLocation(tt.location)
			opt(c)
			if c.location != tt.want {
				t.Errorf("WithLocation() = %v, want %v", c.location, tt.want)
			}
		})
	}
}

func TestWithSize(t *testing.T) {
	tests := []struct {
		name string
		size string
		want string
	}{
		{
			name: "custom size",
			size: "Standard_D2s_v3",
			want: "Standard_D2s_v3",
		},
		{
			name: "empty size defaults to Standard_F2s",
			size: "",
			want: "Standard_F2s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithSize(tt.size)
			opt(c)
			if c.size != tt.want {
				t.Errorf("WithSize() = %v, want %v", c.size, tt.want)
			}
		})
	}
}

func TestWithTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want map[string]string
	}{
		{
			name: "with tags",
			tags: map[string]string{"env": "test", "team": "ci"},
			want: map[string]string{"env": "test", "team": "ci"},
		},
		{
			name: "empty tags",
			tags: map[string]string{},
			want: map[string]string{},
		},
		{
			name: "nil tags",
			tags: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithTags(tt.tags)
			opt(c)
			if len(c.tags) != len(tt.want) {
				t.Errorf("WithTags() length = %v, want %v", len(c.tags), len(tt.want))
			}
			for k, v := range tt.want {
				if c.tags[k] != v {
					t.Errorf("WithTags()[%s] = %v, want %v", k, c.tags[k], v)
				}
			}
		})
	}
}

func TestWithSecurityGroupName(t *testing.T) {
	tests := []struct {
		name              string
		securityGroupName string
		want              string
	}{
		{
			name:              "with security group",
			securityGroupName: "my-nsg",
			want:              "my-nsg",
		},
		{
			name:              "empty security group",
			securityGroupName: "",
			want:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithSecurityGroupName(tt.securityGroupName)
			opt(c)
			if c.securityGroupName != tt.want {
				t.Errorf("WithSecurityGroupName() = %v, want %v", c.securityGroupName, tt.want)
			}
		})
	}
}

func TestWithResourceGroupName(t *testing.T) {
	tests := []struct {
		name              string
		resourceGroupName string
		want              string
	}{
		{
			name:              "with resource group",
			resourceGroupName: "my-resource-group",
			want:              "my-resource-group",
		},
		{
			name:              "empty resource group",
			resourceGroupName: "",
			want:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithResourceGroupName(tt.resourceGroupName)
			opt(c)
			if c.resourceGroupName != tt.want {
				t.Errorf("WithResourceGroupName() = %v, want %v", c.resourceGroupName, tt.want)
			}
		})
	}
}

func TestWithImage(t *testing.T) {
	c := &config{}
	opt := WithImage("MicrosoftWindowsServer", "WindowsServer", "2022-datacenter-g2", "latest")
	opt(c)

	if c.publisher != "MicrosoftWindowsServer" {
		t.Errorf("WithImage() publisher = %v, want %v", c.publisher, "MicrosoftWindowsServer")
	}
	if c.offer != "WindowsServer" {
		t.Errorf("WithImage() offer = %v, want %v", c.offer, "WindowsServer")
	}
	if c.sku != "2022-datacenter-g2" {
		t.Errorf("WithImage() sku = %v, want %v", c.sku, "2022-datacenter-g2")
	}
	if c.version != "latest" {
		t.Errorf("WithImage() version = %v, want %v", c.version, "latest")
	}
}

func TestWithCredentials(t *testing.T) {
	c := &config{}

	WithClientID("test-client-id")(c)
	WithClientSecret("test-client-secret")(c)
	WithSubscriptionID("test-subscription-id")(c)
	WithTenantID("test-tenant-id")(c)

	if c.clientID != "test-client-id" {
		t.Errorf("WithClientID() = %v, want %v", c.clientID, "test-client-id")
	}
	if c.clientSecret != "test-client-secret" {
		t.Errorf("WithClientSecret() = %v, want %v", c.clientSecret, "test-client-secret")
	}
	if c.subscriptionID != "test-subscription-id" {
		t.Errorf("WithSubscriptionID() = %v, want %v", c.subscriptionID, "test-subscription-id")
	}
	if c.tenantID != "test-tenant-id" {
		t.Errorf("WithTenantID() = %v, want %v", c.tenantID, "test-tenant-id")
	}
}

func TestWithUsername(t *testing.T) {
	c := &config{}
	opt := WithUsername("admin")
	opt(c)

	if c.username != "admin" {
		t.Errorf("WithUsername() = %v, want %v", c.username, "admin")
	}
}

func TestWithPassword(t *testing.T) {
	c := &config{}
	opt := WithPassword("secretpassword")
	opt(c)

	if c.password != "secretpassword" {
		t.Errorf("WithPassword() = %v, want %v", c.password, "secretpassword")
	}
}

func TestWithID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "with custom image ID",
			id:   "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Compute/images/my-image",
			want: "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Compute/images/my-image",
		},
		{
			name: "empty ID",
			id:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithID(tt.id)
			opt(c)
			if c.id != tt.want {
				t.Errorf("WithID() = %v, want %v", c.id, tt.want)
			}
		})
	}
}

func TestWithSecurityType(t *testing.T) {
	tests := []struct {
		name         string
		securityType string
		want         string
	}{
		{
			name:         "TrustedLaunch",
			securityType: "TrustedLaunch",
			want:         "TrustedLaunch",
		},
		{
			name:         "empty security type",
			securityType: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			opt := WithSecurityType(tt.securityType)
			opt(c)
			if c.securityType != tt.want {
				t.Errorf("WithSecurityType() = %v, want %v", c.securityType, tt.want)
			}
		})
	}
}

func TestPrivateIPConfigCombinations(t *testing.T) {
	tests := []struct {
		name               string
		privateIP          bool
		vnetName           string
		subnetName         string
		existingSubnetID   string
		wantPrivateIP      bool
		wantUseExistingNet bool
	}{
		{
			name:               "private IP with existing network",
			privateIP:          true,
			vnetName:           "existing-vnet",
			subnetName:         "existing-subnet",
			wantPrivateIP:      true,
			wantUseExistingNet: true,
		},
		{
			name:               "private IP with existing subnet id",
			privateIP:          true,
			existingSubnetID:   "/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s",
			wantPrivateIP:      true,
			wantUseExistingNet: true,
		},
		{
			name:               "private IP with new network (auto-created)",
			privateIP:          true,
			vnetName:           "",
			subnetName:         "",
			wantPrivateIP:      true,
			wantUseExistingNet: false,
		},
		{
			name:               "public IP with existing network",
			privateIP:          false,
			vnetName:           "existing-vnet",
			subnetName:         "existing-subnet",
			wantPrivateIP:      false,
			wantUseExistingNet: true,
		},
		{
			name:               "public IP with new network (default behavior)",
			privateIP:          false,
			vnetName:           "",
			subnetName:         "",
			wantPrivateIP:      false,
			wantUseExistingNet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{}
			WithPrivateIP(tt.privateIP)(c)
			WithVNet(tt.vnetName)(c)
			WithSubnet(tt.subnetName)(c)
			WithExistingSubnetID(tt.existingSubnetID)(c)

			if c.privateIP != tt.wantPrivateIP {
				t.Errorf("privateIP = %v, want %v", c.privateIP, tt.wantPrivateIP)
			}

			if c.useExistingNetwork() != tt.wantUseExistingNet {
				t.Errorf("useExistingNetwork = %v, want %v", c.useExistingNetwork(), tt.wantUseExistingNet)
			}
		})
	}
}

// strPtr is a helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
