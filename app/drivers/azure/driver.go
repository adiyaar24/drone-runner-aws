package azure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/drone/runner-go/logger"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/app/lehelper"
	"github.com/drone-runners/drone-runner-aws/app/oshelp"
	ierrors "github.com/drone-runners/drone-runner-aws/app/types"
	"github.com/drone-runners/drone-runner-aws/command/harness/storage"
	"github.com/drone-runners/drone-runner-aws/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/dchest/uniuri"
)

var _ drivers.Driver = (*config)(nil)

type config struct {
	tenantID          string
	clientID          string
	clientSecret      string
	subscriptionID    string
	resourceGroupName string

	securityGroupName string

	rootDir      string
	id           string
	securityType string

	location string // region, example: East US

	// image data
	publisher string
	offer     string
	sku       string
	version   string

	IPAddress string

	size        string
	tags        map[string]string
	zones       []*string
	userData    string
	userDataKey string

	username string
	password string

	// network configuration
	privateIP         bool   // if true, don't create public IP
	vnetName          string // existing VNet name (optional)
	subnetName        string // existing subnet name (optional)
	existingSubnetID  string // full ARM ID of existing subnet (optional)
	vnetResourceGroup string // RG containing the VNet when using name-based lookup (optional)

	service *armcompute.VirtualMachinesClient
	cred    azcore.TokenCredential
}

func New(opts ...Option) (drivers.Driver, error) {
	p := new(config)
	for _, opt := range opts {
		opt(p)
	}

	if p.tenantID == "" || p.clientID == "" || p.clientSecret == "" || p.subscriptionID == "" {
		return nil, errors.New("missing required azure account credentials (tenant_id, client_id, client_secret, subscription_id)")
	}
	if p.service == nil {
		cred, err := azidentity.NewClientSecretCredential(p.tenantID, p.clientID, p.clientSecret, nil)
		p.cred = cred
		if err != nil {
			return nil, err
		}

		p.service, err = armcompute.NewVirtualMachinesClient(p.subscriptionID, cred, nil)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// useExistingNetwork is true when the NIC should use an existing subnet (by ID or by vnet+subnet name).
func (c *config) useExistingNetwork() bool {
	if c.existingSubnetID != "" {
		return true
	}
	return c.vnetName != "" && c.subnetName != ""
}

func (c *config) RootDir() string {
	return c.rootDir
}

func (c *config) DriverName() string {
	return string(types.Azure)
}

func (c *config) InstanceType() string {
	return c.offer
}

func (c *config) CanHibernate() bool {
	return false
}

func (c *config) GetFullyQualifiedImage(_ context.Context, config *types.VMImageConfig) (string, error) {
	return c.offer, nil
}

func (c *config) Zones() string {
	var z string
	if len(z) == 1 {
		return *c.zones[0]
	}
	for _, zone := range c.zones {
		z = fmt.Sprintf("%s%s,", z, *zone)
	}

	return z
}

// ReserveCapacity reserves capacity for a VM
func (c *config) ReserveCapacity(ctx context.Context, opts *types.InstanceCreateOpts) (*types.CapacityReservation, error) {
	return nil, &ierrors.ErrCapacityReservationNotSupported{Driver: c.DriverName()}
}

// DestroyCapacity destroys capacity for a VM
func (c *config) DestroyCapacity(ctx context.Context, capacity *types.CapacityReservation) (err error) {
	return &ierrors.ErrCapacityReservationNotSupported{Driver: c.DriverName()}
}

func (c *config) Create(ctx context.Context, opts *types.InstanceCreateOpts) (instance *types.Instance, err error) {
	sanitizedRunnerName := strings.ReplaceAll(opts.RunnerName, " ", "-")
	sanitizedPoolName := strings.ReplaceAll(opts.PoolName, " ", "-")
	var name = fmt.Sprintf("%s-%s-%s", sanitizedRunnerName, sanitizedPoolName, uniuri.NewLen(8)) //nolint:mnd
	vnetName := fmt.Sprintf("%s-vnet", name)
	subnetName := fmt.Sprintf("%s-subnet", name)
	publicIPName := fmt.Sprintf("%s-publicip", name)
	networkInterfaceName := fmt.Sprintf("%s-networkinterface", name)
	diskName := fmt.Sprintf("%s-disk", name)

	// Use existing VNet/Subnet if provided (by ARM subnet ID or by vnet + subnet name).
	useExistingNetwork := c.useExistingNetwork()
	if useExistingNetwork && c.existingSubnetID == "" {
		vnetName = c.vnetName
		subnetName = c.subnetName
	}

	var tags = map[string]*string{}
	// add user defined tags
	for k, v := range c.tags {
		tagValue := v
		tags[k] = &tagValue
	}

	logr := logger.FromContext(ctx).
		WithField("cloud", types.Azure).
		WithField("name", name).
		WithField("image", c.InstanceType()).
		WithField("pool", opts.PoolName).
		WithField("zone", c.zones).
		WithField("image", c.offer).
		WithField("size", c.size).
		WithField("private_ip", c.privateIP)

	logr.Info("starting Azure Setup")

	_, err = c.createResourceGroup(ctx)
	if err != nil {
		logr.WithError(err).Errorln("failed to get/create resource group")
		return
	}

	var subnetID string
	if useExistingNetwork {
		if c.existingSubnetID != "" {
			subnetID = c.existingSubnetID
			logr.Debugf("using existing subnet ID: %s", subnetID)
		} else {
			vnetRG := c.resourceGroupName
			if c.vnetResourceGroup != "" {
				vnetRG = c.vnetResourceGroup
			}
			subnet, subnetErr := c.getExistingSubnet(ctx, vnetRG, vnetName, subnetName)
			if subnetErr != nil {
				logr.WithError(subnetErr).Error("could not get existing subnet")
				return nil, subnetErr
			}
			if subnet.ID == nil {
				err = errors.New("existing subnet has nil ID")
				logr.WithError(err).Error("could not get subnet ID")
				return nil, err
			}
			subnetID = *subnet.ID
			logr.Debugf("using existing VNet: %s, Subnet: %s (resource group: %s)", vnetName, subnetName, vnetRG)
		}
	} else {
		// Create new VNet and Subnet
		_, err = c.createVirtualNetwork(ctx, vnetName)
		if err != nil {
			logr.WithError(err).Error("could not create virtual network")
			return nil, err
		}
		subnet, subnetErr := c.createSubnets(ctx, subnetName, vnetName)
		if subnetErr != nil {
			logr.WithError(subnetErr).Error("could not create subnet")
			return nil, subnetErr
		}
		if subnet.ID == nil {
			err = errors.New("created subnet has nil ID")
			logr.WithError(err).Error("could not get subnet ID")
			return nil, err
		}
		subnetID = *subnet.ID
	}

	var networkInterface *armnetwork.Interface
	if c.privateIP {
		// Create network interface without public IP
		networkInterface, err = c.createNetworkInterfacePrivate(ctx, networkInterfaceName, subnetID)
		if err != nil {
			logr.WithError(err).Error("could not create network interface (private)")
			return nil, err
		}
		// Get private IP from the network interface
		c.IPAddress = c.getPrivateIPFromInterface(networkInterface)
		if c.IPAddress == "" {
			err = errors.New("failed to get private IP address from network interface")
			logr.WithError(err).Error("could not obtain private IP")
			return nil, err
		}
		logr.Infof("azure: using private IP: %s", c.IPAddress)
	} else {
		// Create public IP and network interface with public IP
		publicIP, publicIPErr := c.createPublicIP(ctx, publicIPName)
		if publicIPErr != nil {
			logr.WithError(publicIPErr).Error("could not create public IP")
			return nil, publicIPErr
		}
		// Get public IP address with nil check
		if publicIP.Properties == nil || publicIP.Properties.IPAddress == nil {
			err = errors.New("failed to get public IP address: properties or IP address is nil")
			logr.WithError(err).Error("could not obtain public IP")
			return nil, err
		}
		c.IPAddress = *publicIP.Properties.IPAddress
		networkInterface, err = c.createNetworkInterface(ctx, networkInterfaceName, subnetID, *publicIP.ID)
		if err != nil {
			logr.WithError(err).Error("could not create network interface")
			return nil, err
		}
	}

	// create the instance
	startTime := time.Now()

	userData, err := lehelper.GenerateUserdata(c.userData, opts)
	if err != nil {
		logr.WithError(err).
			Errorln("azure: failed to generate user data")
		return nil, err
	}
	uData := base64.StdEncoding.EncodeToString([]byte(userData))

	logr.Traceln("azure: creating VM")
	var imageReference *armcompute.ImageReference
	if c.id != "" {
		imageReference = &armcompute.ImageReference{
			ID: to.Ptr(c.id),
		}
	} else {
		imageReference = &armcompute.ImageReference{
			Publisher: to.Ptr(c.publisher),
			Offer:     to.Ptr(c.offer),
			SKU:       to.Ptr(c.sku),
			Version:   to.Ptr(c.version),
		}
	}

	var in = armcompute.VirtualMachine{
		Location: to.Ptr(c.location),
		Zones:    c.zones,
		Tags:     tags,
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(c.size)),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: imageReference,
				OSDisk: &armcompute.OSDisk{
					Name:         to.Ptr(diskName),
					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
					Caching:      to.Ptr(armcompute.CachingTypesReadWrite),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardLRS), // OSDisk type Standard/Premium HDD/SSD
					},
				},
			},
			OSProfile: &armcompute.OSProfile{ //
				ComputerName:             to.Ptr("vm-runner"),
				AdminUsername:            to.Ptr(c.username),
				AdminPassword:            to.Ptr(c.password),
				CustomData:               to.Ptr(uData),
				AllowExtensionOperations: to.Ptr(true),
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr(*networkInterface.ID),
					},
				},
			},
		},
	}

	if c.securityType != "" {
		securityProfile := &armcompute.SecurityProfile{
			SecurityType: (*armcompute.SecurityTypes)(&c.securityType),
		}
		in.Properties.SecurityProfile = securityProfile
	}

	poller, err := c.service.BeginCreateOrUpdate(ctx, c.resourceGroupName, name, in, nil)
	if err != nil {
		return nil, err
	}
	vm, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	// if windows add extension to vm
	if opts.OS == oshelp.OSWindows {
		_, extensionErr := c.addExtension(ctx, name)
		if extensionErr != nil {
			return nil, extensionErr
		}
	}

	logr.Debugln("instance insert operation completed")

	logr.
		WithField("ip", vm.ID).
		WithField("time", fmt.Sprintf("%.2fs", time.Since(startTime).Seconds())).
		Debugln("azure: [provision] VM provisioned")

	instanceMap := c.mapToInstance(&vm, opts)
	logr.
		WithField("ip", instanceMap.Address).
		WithField("time", fmt.Sprintf("%.2fs", time.Since(startTime).Seconds())).
		Debugln("azure: [provision] complete")

	return &instanceMap, nil
}

func (c *config) Destroy(ctx context.Context, instances []*types.Instance) (err error) {
	return c.DestroyInstanceAndStorage(ctx, instances, nil)
}

func (c *config) DestroyInstanceAndStorage(ctx context.Context, instances []*types.Instance, _ *storage.CleanupType) (err error) {
	var instanceIDs []string
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.ID)
	}
	logr := logger.FromContext(ctx).
		WithField("cloud", types.Azure).
		WithField("image", c.InstanceType()).
		WithField("zone", c.zones).
		WithField("image", c.offer).
		WithField("size", c.size)

	if c.resourceGroupName == "" {
		c.resourceGroupName = defaultResourceGroup
	}
	if len(instanceIDs) == 0 {
		return nil
	}

	useExistingNetwork := c.useExistingNetwork()

	for _, instanceID := range instanceIDs {
		vnetName := fmt.Sprintf("%s-vnet", instanceID)
		publicIPName := fmt.Sprintf("%s-publicip", instanceID)
		networkInterfaceName := fmt.Sprintf("%s-networkinterface", instanceID)
		diskName := fmt.Sprintf("%s-disk", instanceID)

		poller, err := c.service.BeginDelete(ctx, c.resourceGroupName, instanceID, nil)
		if err != nil {
			return err
		}
		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return err
		}
		logr.Info("azure: begin delete VM")
		err = c.deleteNetworkInterface(ctx, networkInterfaceName)
		if err != nil {
			logr.Errorln(err)
			return err
		}
		logr.Info("azure: deleted network interface: ", networkInterfaceName)

		// Only delete public IP if we're not using private IP mode
		if !c.privateIP {
			err = c.deletePublicIP(ctx, publicIPName)
			if err != nil {
				logr.Errorln(err)
				return err
			}
			logr.Info("azure: deleted public ip: ", publicIPName)
		}

		// Only delete VNet if we're not using an existing network
		if !useExistingNetwork {
			err = c.deleteVirtualNetWork(ctx, vnetName)
			if err != nil {
				logr.Errorln(err)
				return err
			}
			logr.Info("azure: deleted virtual network: ", vnetName)
		}
		err = c.deleteDisk(ctx, diskName)
		if err != nil {
			logr.Errorln(err)
			return err
		}
		logr.Info("azure: deleted disk: ", diskName)
		logr.Info("azure: VM deleted")
	}
	return nil
}

func (c *config) Hibernate(_ context.Context, _, _, _ string) error {
	return errors.New("unimplemented")
}

func (c *config) Start(_ context.Context, _ *types.Instance, _ string) (ipAddress string, err error) {
	return "", errors.New("unimplemented")
}

func (c *config) Ping(ctx context.Context) error {
	_, err := azidentity.NewClientSecretCredential(c.tenantID, c.clientID, c.clientSecret, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *config) Logs(ctx context.Context, instanceID string) (string, error) {
	return "", nil
}

func (c *config) SetTags(ctx context.Context, instance *types.Instance,
	tags map[string]string) error {
	return nil
}

func (c *config) mapToInstance(vm *armcompute.VirtualMachinesClientCreateOrUpdateResponse, opts *types.InstanceCreateOpts) types.Instance {
	return types.Instance{
		ID:           *vm.Name,
		Name:         *vm.Name,
		Provider:     types.Azure,
		State:        types.StateProvisioning,
		Pool:         opts.PoolName,
		Image:        c.offer,
		Zone:         c.Zones(),
		Size:         c.size,
		Platform:     opts.Platform,
		Address:      c.IPAddress,
		CACert:       opts.CACert,
		CAKey:        opts.CAKey,
		TLSCert:      opts.TLSCert,
		TLSKey:       opts.TLSKey,
		Started:      vm.Properties.TimeCreated.Unix(),
		Updated:      time.Now().Unix(),
		IsHibernated: false,
		Port:         lehelper.LiteEnginePort,
	}
}
