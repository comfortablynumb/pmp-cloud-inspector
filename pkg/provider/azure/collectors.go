//go:build azure
// +build azure

package azure

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectResourceGroups collects all Azure resource groups
func (p *Provider) collectResourceGroups(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure resource groups...\n")

	client, err := armresources.NewResourceGroupsClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create resource groups client: %w", err)
	}

	pager := client.NewListPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list resource groups: %w", err)
		}

		for _, rg := range page.Value {
			if rg.ID == nil || rg.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":           getString(rg.Location),
				"provisioning_state": getString(rg.Properties.ProvisioningState),
			}

			if rg.Tags != nil {
				tags := make(map[string]string)
				for k, v := range rg.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *rg.ID,
				Type:       resource.TypeAzureResourceGroup,
				Name:       *rg.Name,
				Provider:   "azure",
				Region:     getString(rg.Location),
				Properties: properties,
				RawData:    rg,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d resource groups\n", count)
	return nil
}

// collectVirtualMachines collects all Azure virtual machines
func (p *Provider) collectVirtualMachines(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure virtual machines...\n")

	client, err := armcompute.NewVirtualMachinesClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create VMs client: %w", err)
	}

	pager := client.NewListAllPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			if vm.ID == nil || vm.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":           getString(vm.Location),
				"provisioning_state": getString(vm.Properties.ProvisioningState),
				"resource_group":     extractResourceGroup(*vm.ID),
			}

			if vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
				properties["vm_size"] = string(*vm.Properties.HardwareProfile.VMSize)
			}

			if vm.Properties.OSProfile != nil {
				properties["computer_name"] = getString(vm.Properties.OSProfile.ComputerName)
			}

			if vm.Tags != nil {
				tags := make(map[string]string)
				for k, v := range vm.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *vm.ID,
				Type:       resource.TypeAzureVM,
				Name:       *vm.Name,
				Provider:   "azure",
				Region:     getString(vm.Location),
				Properties: properties,
				RawData:    vm,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d virtual machines\n", count)
	return nil
}

// collectVirtualNetworks collects all Azure virtual networks
func (p *Provider) collectVirtualNetworks(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure virtual networks...\n")

	client, err := armnetwork.NewVirtualNetworksClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create VNets client: %w", err)
	}

	pager := client.NewListAllPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list VNets: %w", err)
		}

		for _, vnet := range page.Value {
			if vnet.ID == nil || vnet.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":       getString(vnet.Location),
				"resource_group": extractResourceGroup(*vnet.ID),
			}

			if vnet.Properties != nil && vnet.Properties.AddressSpace != nil {
				properties["address_prefixes"] = vnet.Properties.AddressSpace.AddressPrefixes
			}

			if vnet.Tags != nil {
				tags := make(map[string]string)
				for k, v := range vnet.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *vnet.ID,
				Type:       resource.TypeAzureVNet,
				Name:       *vnet.Name,
				Provider:   "azure",
				Region:     getString(vnet.Location),
				Properties: properties,
				RawData:    vnet,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d virtual networks\n", count)
	return nil
}

// collectStorageAccounts collects all Azure storage accounts
func (p *Provider) collectStorageAccounts(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure storage accounts...\n")

	client, err := armstorage.NewAccountsClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage accounts client: %w", err)
	}

	pager := client.NewListPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list storage accounts: %w", err)
		}

		for _, sa := range page.Value {
			if sa.ID == nil || sa.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":       getString(sa.Location),
				"resource_group": extractResourceGroup(*sa.ID),
				"kind":           string(*sa.Kind),
				"sku":            string(*sa.SKU.Name),
			}

			if sa.Properties != nil {
				properties["provisioning_state"] = string(*sa.Properties.ProvisioningState)
				properties["primary_location"] = getString(sa.Properties.PrimaryLocation)
			}

			if sa.Tags != nil {
				tags := make(map[string]string)
				for k, v := range sa.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *sa.ID,
				Type:       resource.TypeAzureStorageAccount,
				Name:       *sa.Name,
				Provider:   "azure",
				Region:     getString(sa.Location),
				Properties: properties,
				RawData:    sa,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d storage accounts\n", count)
	return nil
}

// collectAppServices collects all Azure App Services
func (p *Provider) collectAppServices(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure App Services...\n")

	client, err := armappservice.NewWebAppsClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create app services client: %w", err)
	}

	pager := client.NewListPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list app services: %w", err)
		}

		for _, app := range page.Value {
			if app.ID == nil || app.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":       getString(app.Location),
				"resource_group": extractResourceGroup(*app.ID),
				"kind":           getString(app.Kind),
				"state":          getString(app.Properties.State),
			}

			if app.Properties != nil {
				properties["default_hostname"] = getString(app.Properties.DefaultHostName)
				properties["enabled"] = *app.Properties.Enabled
			}

			if app.Tags != nil {
				tags := make(map[string]string)
				for k, v := range app.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *app.ID,
				Type:       resource.TypeAzureAppService,
				Name:       *app.Name,
				Provider:   "azure",
				Region:     getString(app.Location),
				Properties: properties,
				RawData:    app,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d app services\n", count)
	return nil
}

// collectSQLDatabases collects all Azure SQL databases
func (p *Provider) collectSQLDatabases(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure SQL databases...\n")

	// First, list all SQL servers
	serversClient, err := armsql.NewServersClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create SQL servers client: %w", err)
	}

	dbClient, err := armsql.NewDatabasesClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create SQL databases client: %w", err)
	}

	serversPager := serversClient.NewListPager(nil)
	count := 0

	for serversPager.More() {
		serversPage, err := serversPager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list SQL servers: %w", err)
		}

		for _, server := range serversPage.Value {
			if server.Name == nil {
				continue
			}

			rgName := extractResourceGroup(*server.ID)
			if rgName == "" {
				continue
			}

			// List databases for this server
			dbPager := dbClient.NewListByServerPager(rgName, *server.Name, nil)

			for dbPager.More() {
				dbPage, err := dbPager.NextPage(ctx)
				if err != nil {
					fmt.Fprintf(os.Stderr, "    Warning: failed to list databases for server %s: %v\n", *server.Name, err)
					continue
				}

				for _, db := range dbPage.Value {
					if db.ID == nil || db.Name == nil {
						continue
					}

					// Skip master database
					if *db.Name == "master" {
						continue
					}

					properties := map[string]interface{}{
						"location":       getString(db.Location),
						"resource_group": rgName,
						"server_name":    *server.Name,
					}

					if db.Properties != nil && db.Properties.Status != nil {
						properties["status"] = string(*db.Properties.Status)
					}

					if db.SKU != nil {
						properties["sku"] = *db.SKU.Name
					}

					if db.Tags != nil {
						tags := make(map[string]string)
						for k, v := range db.Tags {
							if v != nil {
								tags[k] = *v
							}
						}
						properties["tags"] = tags
					}

					res := &resource.Resource{
						ID:         *db.ID,
						Type:       resource.TypeAzureSQLDatabase,
						Name:       *db.Name,
						Provider:   "azure",
						Region:     getString(db.Location),
						Properties: properties,
						RawData:    db,
					}

					collection.Add(res)
					count++
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d SQL databases\n", count)
	return nil
}

// collectKeyVaults collects all Azure Key Vaults
func (p *Provider) collectKeyVaults(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Azure Key Vaults...\n")

	client, err := armkeyvault.NewVaultsClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create key vaults client: %w", err)
	}

	pager := client.NewListPager(nil)
	count := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list key vaults: %w", err)
		}

		for _, kv := range page.Value {
			if kv.ID == nil || kv.Name == nil {
				continue
			}

			properties := map[string]interface{}{
				"location":       getString(kv.Location),
				"resource_group": extractResourceGroup(*kv.ID),
			}

			// Note: List returns Resource type, need to use Get for full details if needed
			// For now, just collect basic information available from list

			if kv.Tags != nil {
				tags := make(map[string]string)
				for k, v := range kv.Tags {
					if v != nil {
						tags[k] = *v
					}
				}
				properties["tags"] = tags
			}

			res := &resource.Resource{
				ID:         *kv.ID,
				Type:       resource.TypeAzureKeyVault,
				Name:       *kv.Name,
				Provider:   "azure",
				Region:     getString(kv.Location),
				Properties: properties,
				RawData:    kv,
			}

			collection.Add(res)
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d key vaults\n", count)
	return nil
}

// Helper functions

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func extractResourceGroup(resourceID string) string {
	// Azure resource ID format: /subscriptions/{sub-id}/resourceGroups/{rg-name}/...
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
