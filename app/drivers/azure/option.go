package azure

import (
	"fmt"
	"os"

	"github.com/drone-runners/drone-runner-aws/app/oshelp"
	"github.com/drone-runners/drone-runner-aws/types"

	"github.com/sirupsen/logrus"
)

type Option func(*config)

func SetPlatformDefaults(platform *types.Platform) (*types.Platform, error) {
	if platform.Arch == "" {
		platform.Arch = oshelp.ArchAMD64
	}
	if platform.Arch != oshelp.ArchAMD64 && platform.Arch != oshelp.ArchARM64 {
		return platform, fmt.Errorf("invalid arch %s, has to be '%s/%s'", platform.Arch, oshelp.ArchAMD64, oshelp.ArchARM64)
	}
	// verify that we are using sane values for OS
	if platform.OS == "" {
		platform.OS = oshelp.OSWindows
	}
	if platform.OS != oshelp.OSWindows {
		return platform, fmt.Errorf("azure - invalid OS %s, has to be '%s'", platform.OS, oshelp.OSWindows)
	}

	return platform, nil
}

func WithClientID(clientID string) Option {
	return func(p *config) {
		p.clientID = clientID
	}
}

func WithClientSecret(clientSecret string) Option {
	return func(p *config) {
		p.clientSecret = clientSecret
	}
}

func WithSubscriptionID(subscriptionID string) Option {
	return func(p *config) {
		p.subscriptionID = subscriptionID
	}
}

func WithTenantID(tenantID string) Option {
	return func(p *config) {
		p.tenantID = tenantID
	}
}

func WithResourceGroupName(resourceGroupName string) Option {
	return func(p *config) {
		p.resourceGroupName = resourceGroupName
	}
}

func WithLocation(location string) Option {
	return func(p *config) {
		if location == "" {
			p.location = "eastus2"
		} else {
			p.location = location
		}
	}
}

func WithSize(size string) Option {
	return func(p *config) {
		if size == "" {
			p.size = "Standard_F2s"
		} else {
			p.size = size
		}
	}
}

func WithImage(publisher, offer, sku, version string) Option {
	return func(p *config) {
		p.publisher = publisher
		p.offer = offer
		p.sku = sku
		p.version = version
	}
}

func WithID(id string) Option {
	return func(p *config) {
		p.id = id
	}
}

func WithSecurityType(securityType string) Option {
	return func(p *config) {
		p.securityType = securityType
	}
}

func WithUsername(username string) Option {
	return func(p *config) {
		p.username = username
	}
}

func WithPassword(password string) Option {
	return func(p *config) {
		p.password = password
	}
}

// WithRootDirectory sets the root directory for the virtual machine.
func WithRootDirectory(dir string) Option {
	return func(p *config) {
		p.rootDir = tempdir(dir)
	}
}

// WithUserData returns an option to set the cloud-init template from a file location or passed in text.
func WithUserData(text, path string) Option {
	if text != "" {
		return func(p *config) {
			p.userData = text
		}
	}
	return func(p *config) {
		if path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				logrus.WithError(err).
					Fatalln("failed to read user_data file")
				return
			}
			p.userData = string(data)
		}
	}
}

// WithUserDataKey allows to set the user data key for Google Cloud Platform
// This allows user to set either user-data or a startup script
func WithUserDataKey(text, platform string) Option {
	return func(p *config) {
		p.userDataKey = text
		if p.userDataKey == "" && platform == oshelp.OSLinux {
			p.userDataKey = "user-data"
		} else {
			p.userDataKey = "windows-startup-script-ps1"
		}
	}
}

func WithZones(zones ...string) Option {
	var z []*string
	if len(zones) > 0 {
		zone1 := "1"
		z = append(z, &zone1)
	} else {
		for zone := range zones {
			z = append(z, &zones[zone])
		}
	}
	return func(p *config) {
		p.zones = z
	}
}

// WithTags returns an option to set the resource tags.
func WithTags(t map[string]string) Option {
	return func(p *config) {
		p.tags = t
	}
}

func WithSecurityGroupName(securityGroupName string) Option {
	return func(p *config) {
		p.securityGroupName = securityGroupName
	}
}

// WithPrivateIP returns an option to use private IP addresses only (no public IP).
func WithPrivateIP(private bool) Option {
	return func(p *config) {
		p.privateIP = private
	}
}

// WithVNet returns an option to use an existing Virtual Network.
func WithVNet(vnetName string) Option {
	return func(p *config) {
		p.vnetName = vnetName
	}
}

// WithSubnet returns an option to use an existing Subnet.
func WithSubnet(subnetName string) Option {
	return func(p *config) {
		p.subnetName = subnetName
	}
}

// WithExistingSubnetID returns an option to attach the NIC to an existing subnet by full ARM resource ID.
func WithExistingSubnetID(subnetID string) Option {
	return func(p *config) {
		p.existingSubnetID = subnetID
	}
}

// WithVNetResourceGroup returns the resource group name that contains the VNet (when using vnet/subnet by name).
func WithVNetResourceGroup(resourceGroup string) Option {
	return func(p *config) {
		p.vnetResourceGroup = resourceGroup
	}
}
