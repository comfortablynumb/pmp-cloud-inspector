package github

import (
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// discoverOrganizationRelationships discovers relationships for organizations
func (p *Provider) discoverOrganizationRelationships(org *resource.Resource, collection *resource.Collection) {
	// Find all repositories in this organization
	for _, res := range collection.Resources {
		if res.Type == resource.TypeGitHubRepository {
			for _, rel := range res.Relationships {
				if rel.TargetID == org.ID && rel.Type == resource.RelationBelongsTo {
					// Add inverse relationship
					org.Relationships = append(org.Relationships, resource.Relationship{
						Type:       resource.RelationContains,
						TargetID:   res.ID,
						TargetType: resource.TypeGitHubRepository,
					})
				}
			}
		}

		// Find all teams in this organization
		if res.Type == resource.TypeGitHubTeam {
			for _, rel := range res.Relationships {
				if rel.TargetID == org.ID && rel.Type == resource.RelationBelongsTo {
					// Add inverse relationship
					org.Relationships = append(org.Relationships, resource.Relationship{
						Type:       resource.RelationContains,
						TargetID:   res.ID,
						TargetType: resource.TypeGitHubTeam,
					})
				}
			}
		}

		// Find all users in this organization
		if res.Type == resource.TypeGitHubUser && res.Account == org.Name {
			org.Relationships = append(org.Relationships, resource.Relationship{
				Type:       resource.RelationContains,
				TargetID:   res.ID,
				TargetType: resource.TypeGitHubUser,
			})
		}
	}
}

// discoverRepositoryRelationships discovers relationships for repositories
func (p *Provider) discoverRepositoryRelationships(repo *resource.Resource, collection *resource.Collection) {
	// Repository relationships are already added during collection
	// (belongs_to Organization relationship)
}

// discoverTeamRelationships discovers relationships for teams
func (p *Provider) discoverTeamRelationships(team *resource.Resource, collection *resource.Collection) {
	// Team relationships are already added during collection
	// (belongs_to Organization relationship)

	// Could add more complex relationships here, such as:
	// - Team members (User resources)
	// - Team repositories
}
