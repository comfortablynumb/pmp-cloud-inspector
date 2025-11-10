//go:build gcp
// +build gcp

package gcp

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/functions/apiv1/functionspb"
	"cloud.google.com/go/run/apiv2/runpb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectComputeInstances collects compute instances in a region
func (p *Provider) collectComputeInstances(ctx context.Context, collection *resource.Collection, region string) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP compute instances in %s...\n", region)

	// List zones in the region (simplified: use common zones)
	zones := []string{
		fmt.Sprintf("%s-a", region),
		fmt.Sprintf("%s-b", region),
		fmt.Sprintf("%s-c", region),
	}

	count := 0
	for _, zone := range zones {
		req := &computepb.ListInstancesRequest{
			Project: p.projectID,
			Zone:    zone,
		}

		it := p.computeClient.List(ctx, req)
		for {
			inst, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				// Zone might not exist, continue to next
				break
			}

			res := p.convertComputeInstanceToResource(inst, zone)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found instance: %s (%s)\n", safeString(inst.Name), safeString(inst.Status))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d compute instances in %s\n", count, region)
	return nil
}

// collectNetworks collects VPC networks
func (p *Provider) collectNetworks(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP VPC networks...\n")

	req := &computepb.ListNetworksRequest{
		Project: p.projectID,
	}

	it := p.networksClient.List(ctx, req)
	count := 0
	for {
		network, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list networks: %w", err)
		}

		res := p.convertNetworkToResource(network)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found VPC network: %s\n", safeString(network.Name))
	}

	fmt.Fprintf(os.Stderr, "  Collected %d VPC networks\n", count)
	return nil
}

// collectSubnetworks collects subnetworks in a region
func (p *Provider) collectSubnetworks(ctx context.Context, collection *resource.Collection, region string) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP subnetworks in %s...\n", region)

	req := &computepb.ListSubnetworksRequest{
		Project: p.projectID,
		Region:  region,
	}

	it := p.subnetworksClient.List(ctx, req)
	count := 0
	for {
		subnet, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list subnetworks: %w", err)
		}

		res := p.convertSubnetworkToResource(subnet, region)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found subnetwork: %s (%s)\n", safeString(subnet.Name), safeString(subnet.IpCidrRange))
	}

	fmt.Fprintf(os.Stderr, "  Collected %d subnetworks in %s\n", count, region)
	return nil
}

// collectStorageBuckets collects storage buckets
func (p *Provider) collectStorageBuckets(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP storage buckets...\n")

	it := p.storageClient.Buckets(ctx, p.projectID)
	count := 0
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list buckets: %w", err)
		}

		res := p.convertStorageBucketToResource(attrs)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found storage bucket: %s (%s)\n", attrs.Name, attrs.Location)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d storage buckets\n", count)
	return nil
}

// collectCloudFunctions collects Cloud Functions in a region
func (p *Provider) collectCloudFunctions(ctx context.Context, collection *resource.Collection, region string) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP Cloud Functions in %s...\n", region)

	req := &functionspb.ListFunctionsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", p.projectID, region),
	}

	it := p.functionsClient.ListFunctions(ctx, req)
	count := 0
	for {
		fn, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Region might not support Cloud Functions
			break
		}

		res := p.convertCloudFunctionToResource(fn, region)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found Cloud Function: %s\n", fn.Name)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d Cloud Functions in %s\n", count, region)
	return nil
}

// collectCloudRunServices collects Cloud Run services in a region
func (p *Provider) collectCloudRunServices(ctx context.Context, collection *resource.Collection, region string) error {
	fmt.Fprintf(os.Stderr, "  Collecting GCP Cloud Run services in %s...\n", region)

	req := &runpb.ListServicesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", p.projectID, region),
	}

	it := p.runClient.ListServices(ctx, req)
	count := 0
	for {
		svc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Region might not support Cloud Run
			break
		}

		res := p.convertCloudRunServiceToResource(svc, region)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found Cloud Run service: %s\n", svc.Name)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d Cloud Run services in %s\n", count, region)
	return nil
}

// Conversion functions

func (p *Provider) convertComputeInstanceToResource(inst *computepb.Instance, zone string) *resource.Resource {
	properties := map[string]interface{}{
		"machine_type": safeString(inst.MachineType),
		"status":       safeString(inst.Status),
		"zone":         zone,
	}

	if inst.CpuPlatform != nil {
		properties["cpu_platform"] = *inst.CpuPlatform
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", safeUint64(inst.Id)),
		Type:       resource.TypeGCPComputeInstance,
		Name:       safeString(inst.Name),
		Provider:   "gcp",
		Account:    p.projectID,
		Region:     zone,
		Properties: properties,
		RawData:    inst,
	}

	if inst.CreationTimestamp != nil {
		// Parse timestamp if needed
		properties["creation_timestamp"] = *inst.CreationTimestamp
	}

	return res
}

func (p *Provider) convertNetworkToResource(network *computepb.Network) *resource.Resource {
	properties := map[string]interface{}{
		"auto_create_subnetworks": network.AutoCreateSubnetworks != nil && *network.AutoCreateSubnetworks,
	}

	if network.Description != nil {
		properties["description"] = *network.Description
	}

	return &resource.Resource{
		ID:         fmt.Sprintf("%d", safeUint64(network.Id)),
		Type:       resource.TypeGCPVPC,
		Name:       safeString(network.Name),
		Provider:   "gcp",
		Account:    p.projectID,
		Properties: properties,
		RawData:    network,
	}
}

func (p *Provider) convertSubnetworkToResource(subnet *computepb.Subnetwork, region string) *resource.Resource {
	properties := map[string]interface{}{
		"ip_cidr_range": safeString(subnet.IpCidrRange),
		"network":       safeString(subnet.Network),
	}

	if subnet.Description != nil {
		properties["description"] = *subnet.Description
	}
	if subnet.PrivateIpGoogleAccess != nil {
		properties["private_ip_google_access"] = *subnet.PrivateIpGoogleAccess
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", safeUint64(subnet.Id)),
		Type:       resource.TypeGCPSubnet,
		Name:       safeString(subnet.Name),
		Provider:   "gcp",
		Account:    p.projectID,
		Region:     region,
		Properties: properties,
		RawData:    subnet,
	}

	// Add network relationship
	if subnet.Network != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *subnet.Network,
			TargetType: resource.TypeGCPVPC,
		})
	}

	return res
}

func (p *Provider) convertStorageBucketToResource(attrs *storage.BucketAttrs) *resource.Resource {
	properties := map[string]interface{}{
		"location":      attrs.Location,
		"storage_class": attrs.StorageClass,
	}

	if attrs.VersioningEnabled {
		properties["versioning_enabled"] = true
	}

	res := &resource.Resource{
		ID:         attrs.Name,
		Type:       resource.TypeGCPStorageBucket,
		Name:       attrs.Name,
		Provider:   "gcp",
		Account:    p.projectID,
		Region:     attrs.Location,
		Properties: properties,
		RawData:    attrs,
	}

	if !attrs.Created.IsZero() {
		res.CreatedAt = &attrs.Created
	}

	return res
}

func (p *Provider) convertCloudFunctionToResource(fn *functionspb.CloudFunction, region string) *resource.Resource {
	properties := map[string]interface{}{
		"runtime":     fn.Runtime,
		"entry_point": fn.EntryPoint,
	}

	if fn.Description != "" {
		properties["description"] = fn.Description
	}
	if fn.AvailableMemoryMb > 0 {
		properties["memory_mb"] = fn.AvailableMemoryMb
	}
	if fn.Timeout != nil {
		properties["timeout"] = fn.Timeout.Seconds
	}

	return &resource.Resource{
		ID:         fn.Name,
		Type:       resource.TypeGCPCloudFunction,
		Name:       fn.Name,
		Provider:   "gcp",
		Account:    p.projectID,
		Region:     region,
		Properties: properties,
		RawData:    fn,
	}
}

func (p *Provider) convertCloudRunServiceToResource(svc *runpb.Service, region string) *resource.Resource {
	properties := map[string]interface{}{
		"uri": svc.Uri,
	}

	if svc.Description != "" {
		properties["description"] = svc.Description
	}

	res := &resource.Resource{
		ID:         svc.Name,
		Type:       resource.TypeGCPCloudRun,
		Name:       svc.Name,
		Provider:   "gcp",
		Account:    p.projectID,
		Region:     region,
		Properties: properties,
		RawData:    svc,
	}

	if svc.CreateTime != nil {
		t := svc.CreateTime.AsTime()
		res.CreatedAt = &t
	}

	return res
}
