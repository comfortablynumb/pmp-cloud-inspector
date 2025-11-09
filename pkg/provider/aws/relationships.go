package aws

import (
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// discoverSubnetRelationships discovers relationships for subnets
func (p *Provider) discoverSubnetRelationships(subnet *resource.Resource, collection *resource.Collection) {
	// Subnet relationships are already added during collection
	// (belongs_to VPC relationship)
}

// discoverSecurityGroupRelationships discovers relationships for security groups
func (p *Provider) discoverSecurityGroupRelationships(sg *resource.Resource, collection *resource.Collection) {
	// Security group relationships are already added during collection
	// (belongs_to VPC relationship)

	// Could add more complex relationships here, such as:
	// - References to other security groups in rules
	// - Associations with specific resources (EC2 instances, etc.)
}

// discoverVPCRelationships discovers relationships for VPCs
func (p *Provider) discoverVPCRelationships(vpc *resource.Resource, collection *resource.Collection) {
	// Find all subnets in this VPC
	for _, res := range collection.Resources {
		if res.Type == resource.TypeAWSSubnet {
			for _, rel := range res.Relationships {
				if rel.TargetID == vpc.ID && rel.Type == resource.RelationBelongsTo {
					// Add inverse relationship
					vpc.Relationships = append(vpc.Relationships, resource.Relationship{
						Type:       resource.RelationContains,
						TargetID:   res.ID,
						TargetType: resource.TypeAWSSubnet,
					})
				}
			}
		}

		// Find all security groups in this VPC
		if res.Type == resource.TypeAWSSecurityGroup {
			for _, rel := range res.Relationships {
				if rel.TargetID == vpc.ID && rel.Type == resource.RelationBelongsTo {
					// Add inverse relationship
					vpc.Relationships = append(vpc.Relationships, resource.Relationship{
						Type:       resource.RelationContains,
						TargetID:   res.ID,
						TargetType: resource.TypeAWSSecurityGroup,
					})
				}
			}
		}
	}
}
