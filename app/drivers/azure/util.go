package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/drone/runner-go/logger"

	"github.com/drone-runners/drone-runner-aws/app/oshelp"
)

const (
	defaultResourceGroup         = "harness-runner-resource"
	defaultVirtualNetworkAddress = "10.1.0.0/16"
	defaultSubnetAddress         = "10.1.10.0/24"
)

func (c *config) createResourceGroup(ctx context.Context) (*armresources.ResourceGroup, error) {
	logr := logger.FromContext(ctx)
	if c.resourceGroupName == "" {
		logr.Debugln("using default resource group name")
		c.resourceGroupName = defaultResourceGroup
	}
	resourceGroupClient, err := armresources.NewResourceGroupsClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}

	parameters := armresources.ResourceGroup{
		Location: to.Ptr(c.location),
	}
	rg, err := resourceGroupClient.Get(ctx, c.resourceGroupName, nil)
	if err != nil {
		resp, createErr := resourceGroupClient.CreateOrUpdate(ctx, c.resourceGroupName, parameters, nil)
		if createErr != nil {
			return nil, createErr
		}
		logr.Debugf("created resource group: %s", c.resourceGroupName)
		return &resp.ResourceGroup, nil
	}
	logr.Debugf("resource group %s already exists", c.resourceGroupName)
	return &rg.ResourceGroup, nil
}

func (c *config) createVirtualNetwork(ctx context.Context, vnetName string) (*armnetwork.VirtualNetwork, error) {
	logr := logger.FromContext(ctx)
	vnetClient, err := armnetwork.NewVirtualNetworksClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}
	parameters := armnetwork.VirtualNetwork{
		Location: to.Ptr(c.location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					to.Ptr(defaultVirtualNetworkAddress), // example 10.1.0.0/16
				},
			},
		},
	}

	pollerResponse, createErr := vnetClient.BeginCreateOrUpdate(ctx, c.resourceGroupName, vnetName, parameters, nil)
	if createErr != nil {
		return nil, createErr
	}

	resp, poolErr := pollerResponse.PollUntilDone(ctx, nil)
	if poolErr != nil {
		return nil, poolErr
	}
	logr.Debugf("created vnet: %s", vnetName)
	return &resp.VirtualNetwork, nil
}

func (c *config) deleteVirtualNetWork(ctx context.Context, vnetName string) error {
	vnetClient, err := armnetwork.NewVirtualNetworksClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return err
	}

	pollerResponse, err := vnetClient.BeginDelete(ctx, c.resourceGroupName, vnetName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResponse.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *config) createSubnets(ctx context.Context, subnetName, vnetName string) (*armnetwork.Subnet, error) {
	logr := logger.FromContext(ctx)
	subnetClient, err := armnetwork.NewSubnetsClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}
	parameters := armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.Ptr(defaultSubnetAddress),
		},
	}
	// map security group to subnet if exists
	if c.securityGroupName != "" {
		securityGroupClient, _ := armnetwork.NewSecurityGroupsClient(c.subscriptionID, c.cred, nil)
		sG, sgErr := securityGroupClient.Get(ctx, c.resourceGroupName, c.securityGroupName, nil)
		if sgErr != nil {
			logr.Infof("failed to get security group %s: %s", c.securityGroupName, sgErr)
			return nil, sgErr
		}
		parameters.Properties.NetworkSecurityGroup = &sG.SecurityGroup
	}

	pollerResponse, createErr := subnetClient.BeginCreateOrUpdate(ctx, c.resourceGroupName, vnetName, subnetName, parameters, nil)
	if createErr != nil {
		return nil, createErr
	}

	resp, pollErr := pollerResponse.PollUntilDone(ctx, nil)
	if pollErr != nil {
		return nil, pollErr
	}
	logr.Debugf("created subnet: %s", subnetName)
	return &resp.Subnet, nil
}

func (c *config) getExistingSubnet(ctx context.Context, vnetResourceGroupName, vnetName, subnetName string) (*armnetwork.Subnet, error) {
	logr := logger.FromContext(ctx)
	subnetClient, err := armnetwork.NewSubnetsClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}

	resp, err := subnetClient.Get(ctx, vnetResourceGroupName, vnetName, subnetName, nil)
	if err != nil {
		return nil, err
	}
	logr.Debugf("using existing subnet: %s in vnet: %s (rg: %s)", subnetName, vnetName, vnetResourceGroupName)
	return &resp.Subnet, nil
}

func (c *config) createPublicIP(ctx context.Context, publicIPName string) (*armnetwork.PublicIPAddress, error) {
	logr := logger.FromContext(ctx)
	publicIPAddressClient, err := armnetwork.NewPublicIPAddressesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}

	parameters := armnetwork.PublicIPAddress{
		Location: to.Ptr(c.location),
		Zones:    c.zones,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic), // Static or Dynamic
		},
	}

	pollerResponse, createErr := publicIPAddressClient.BeginCreateOrUpdate(ctx, c.resourceGroupName, publicIPName, parameters, nil)
	if createErr != nil {
		return nil, createErr
	}

	resp, poolErr := pollerResponse.PollUntilDone(ctx, nil)
	if poolErr != nil {
		return nil, poolErr
	}
	logr.Debugf("created IP: %s", publicIPName)
	return &resp.PublicIPAddress, nil
}

func (c *config) deletePublicIP(ctx context.Context, publicIPName string) error {
	publicIPAddressClient, err := armnetwork.NewPublicIPAddressesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return err
	}

	pollerResponse, err := publicIPAddressClient.BeginDelete(ctx, c.resourceGroupName, publicIPName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResponse.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *config) createNetworkInterface(ctx context.Context, networkInterfaceName, subnetID, publicIPID string) (*armnetwork.Interface, error) {
	logr := logger.FromContext(ctx)
	nicClient, err := armnetwork.NewInterfacesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}

	parameters := armnetwork.Interface{
		Location: to.Ptr(c.location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr("ipConfig"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(subnetID),
						},
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: to.Ptr(publicIPID),
						},
					},
				},
			},
		},
	}

	// Attach NSG to network interface if specified
	if c.securityGroupName != "" {
		securityGroupClient, sgClientErr := armnetwork.NewSecurityGroupsClient(c.subscriptionID, c.cred, nil)
		if sgClientErr == nil {
			sG, sgErr := securityGroupClient.Get(ctx, c.resourceGroupName, c.securityGroupName, nil)
			if sgErr != nil {
				logr.Warnf("failed to get security group %s: %s", c.securityGroupName, sgErr)
			} else {
				parameters.Properties.NetworkSecurityGroup = &armnetwork.SecurityGroup{
					ID: sG.ID,
				}
				logr.Debugf("attaching NSG %s to network interface", c.securityGroupName)
			}
		}
	}

	pollerResponse, createErr := nicClient.BeginCreateOrUpdate(ctx, c.resourceGroupName, networkInterfaceName, parameters, nil)
	if createErr != nil {
		return nil, createErr
	}

	resp, pollErr := pollerResponse.PollUntilDone(ctx, nil)
	if pollErr != nil {
		return nil, pollErr
	}
	logr.Debug("created network interface: %s", networkInterfaceName)
	return &resp.Interface, nil
}

// createNetworkInterfacePrivate creates a network interface without a public IP address
func (c *config) createNetworkInterfacePrivate(ctx context.Context, networkInterfaceName, subnetID string) (*armnetwork.Interface, error) {
	logr := logger.FromContext(ctx)
	nicClient, err := armnetwork.NewInterfacesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return nil, err
	}

	parameters := armnetwork.Interface{
		Location: to.Ptr(c.location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr("ipConfig"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(subnetID),
						},
						// No PublicIPAddress - using private IP only
					},
				},
			},
		},
	}

	// Attach NSG to network interface if specified
	if c.securityGroupName != "" {
		securityGroupClient, sgClientErr := armnetwork.NewSecurityGroupsClient(c.subscriptionID, c.cred, nil)
		if sgClientErr == nil {
			sG, sgErr := securityGroupClient.Get(ctx, c.resourceGroupName, c.securityGroupName, nil)
			if sgErr != nil {
				logr.Warnf("failed to get security group %s: %s", c.securityGroupName, sgErr)
			} else {
				parameters.Properties.NetworkSecurityGroup = &armnetwork.SecurityGroup{
					ID: sG.ID,
				}
				logr.Debugf("attaching NSG %s to network interface (private)", c.securityGroupName)
			}
		}
	}

	pollerResponse, createErr := nicClient.BeginCreateOrUpdate(ctx, c.resourceGroupName, networkInterfaceName, parameters, nil)
	if createErr != nil {
		return nil, createErr
	}

	resp, pollErr := pollerResponse.PollUntilDone(ctx, nil)
	if pollErr != nil {
		return nil, pollErr
	}
	logr.Debugf("created network interface (private IP only): %s", networkInterfaceName)
	return &resp.Interface, nil
}

func (c *config) addExtension(ctx context.Context, vmName string) (ext armcompute.VirtualMachineExtension, err error) {
	// https://github.com/Azure-Samples/azure-sdk-for-go-samples/blob/648055b2a48e67b7f93f0c32411366d8dd3c239a/services/compute/vm_with_identity.go
	logr := logger.FromContext(ctx)
	extension := armcompute.VirtualMachineExtension{
		Location: to.Ptr(c.location),
		Properties: &armcompute.VirtualMachineExtensionProperties{
			Publisher:               to.Ptr("Microsoft.Compute"),
			Type:                    to.Ptr("CustomScriptExtension"),
			TypeHandlerVersion:      to.Ptr("1.0"),
			AutoUpgradeMinorVersion: to.Ptr(true),
			Settings: to.Ptr(map[string]interface{}{
				// EncodedCommand "cp C:\AzureData\CustomData.bin C:\AzureData\CustomData.ps1; C:\AzureData\CustomData.ps1"
				"commandToExecute": `powershell.exe -ExecutionPolicy ByPass -EncodedCommand YwBwACAAQwA6AFwAQQB6AHUAcgBlAEQAYQB0AGEAXABDAHUAcwB0AG8AbQBEAGEAdABhAC4AYgBpAG4AIABDADoAXABBAHoAdQByAGUARABhAHQAYQBcAEMAdQBzAHQAbwBtAEQAYQB0AGEALgBwAHMAMQA7ACAAQwA6AFwAQQB6AHUAcgBlAEQAYQB0AGEAXABDAHUAcwB0AG8AbQBEAGEAdABhAC4AcABzADEA`, //nolint:lll
			}),
		},
	}
	client, clientErr := armcompute.NewVirtualMachineExtensionsClient(c.subscriptionID, c.cred, nil)
	if clientErr != nil {
		return ext, clientErr
	}
	pollerResponse, createErr := client.BeginCreateOrUpdate(ctx, c.resourceGroupName, vmName, "CustomScriptExtension", extension, nil)
	if createErr != nil {
		return ext, createErr
	}
	resp, pollErr := pollerResponse.PollUntilDone(ctx, nil)
	if pollErr != nil {
		return ext, pollErr
	}
	logr.Debug("created extension: CustomScriptExtension for liteengine")
	return resp.VirtualMachineExtension, nil
}

func (c *config) deleteNetworkInterface(ctx context.Context, networkInterfaceName string) error {
	nicClient, err := armnetwork.NewInterfacesClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return err
	}

	pollerResponse, err := nicClient.BeginDelete(ctx, c.resourceGroupName, networkInterfaceName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResponse.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *config) deleteDisk(ctx context.Context, diskName string) error {
	diskClient, err := armcompute.NewDisksClient(c.subscriptionID, c.cred, nil)
	if err != nil {
		return err
	}

	pollerResponse, err := diskClient.BeginDelete(ctx, c.resourceGroupName, diskName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResponse.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

// helper function returns the base temporary directory based on the target platform.
func tempdir(inputOS string) string {
	const dir = "azure"

	switch inputOS {
	case oshelp.OSWindows:
		return oshelp.JoinPaths(inputOS, "C:\\Windows\\Temp", dir)
	default:
		return oshelp.JoinPaths(inputOS, "/tmp", dir)
	}
}

// getPrivateIPFromInterface extracts the private IP address from a network interface.
// Returns empty string if the IP cannot be obtained (similar to AWS/GCP patterns).
func (c *config) getPrivateIPFromInterface(nic *armnetwork.Interface) string {
	if nic == nil {
		return ""
	}
	if nic.Properties == nil {
		return ""
	}
	if len(nic.Properties.IPConfigurations) == 0 {
		return ""
	}
	ipConfig := nic.Properties.IPConfigurations[0]
	// IPConfigurations is a slice of pointers, so the element itself could be nil
	if ipConfig == nil {
		return ""
	}
	if ipConfig.Properties == nil {
		return ""
	}
	if ipConfig.Properties.PrivateIPAddress == nil {
		return ""
	}
	return *ipConfig.Properties.PrivateIPAddress
}
