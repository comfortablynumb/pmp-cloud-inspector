package github

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v57/github"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectOrganizations collects all organizations
func (p *Provider) collectOrganizations(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GitHub organizations...\n")
	count := 0
	for _, orgName := range p.organizations {
		org, _, err := p.client.Organizations.Get(ctx, orgName)
		if err != nil {
			return fmt.Errorf("failed to get organization %s: %w", orgName, err)
		}

		res := p.convertOrganizationToResource(org)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found organization: %s\n", safeString(org.Login))
	}

	fmt.Fprintf(os.Stderr, "  Collected %d organizations\n", count)
	return nil
}

// collectRepositories collects all repositories for an organization
func (p *Provider) collectRepositories(ctx context.Context, collection *resource.Collection, org string) error {
	fmt.Fprintf(os.Stderr, "  Collecting repositories for %s...\n", org)
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	count := 0
	for {
		repos, resp, err := p.client.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}

		for _, repo := range repos {
			res := p.convertRepositoryToResource(repo, org)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found repository: %s/%s\n", org, safeString(repo.Name))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d repositories for %s\n", count, org)
	return nil
}

// collectTeams collects all teams for an organization
func (p *Provider) collectTeams(ctx context.Context, collection *resource.Collection, org string) error {
	fmt.Fprintf(os.Stderr, "  Collecting teams for %s...\n", org)
	opts := &github.ListOptions{PerPage: 100}

	count := 0
	for {
		teams, resp, err := p.client.Teams.ListTeams(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list teams: %w", err)
		}

		for _, team := range teams {
			res := p.convertTeamToResource(team, org)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found team: %s\n", safeString(team.Name))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d teams for %s\n", count, org)
	return nil
}

// collectUsers collects all members for an organization
func (p *Provider) collectUsers(ctx context.Context, collection *resource.Collection, org string) error {
	fmt.Fprintf(os.Stderr, "  Collecting users for %s...\n", org)
	opts := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	count := 0
	for {
		users, resp, err := p.client.Organizations.ListMembers(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list members: %w", err)
		}

		for _, user := range users {
			res := p.convertUserToResource(user, org)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found user: %s\n", safeString(user.Login))
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d users for %s\n", count, org)
	return nil
}

// convertOrganizationToResource converts a GitHub organization to a Resource
func (p *Provider) convertOrganizationToResource(org *github.Organization) *resource.Resource {
	properties := make(map[string]interface{})

	if org.Description != nil {
		properties["description"] = *org.Description
	}
	if org.Company != nil {
		properties["company"] = *org.Company
	}
	if org.Location != nil {
		properties["location"] = *org.Location
	}
	if org.Email != nil {
		properties["email"] = *org.Email
	}
	if org.PublicRepos != nil {
		properties["public_repos"] = *org.PublicRepos
	}
	if org.PublicGists != nil {
		properties["public_gists"] = *org.PublicGists
	}
	if org.Followers != nil {
		properties["followers"] = *org.Followers
	}
	if org.Following != nil {
		properties["following"] = *org.Following
	}

	var createdAt *time.Time
	if org.CreatedAt != nil {
		createdAt = &org.CreatedAt.Time
	}

	var updatedAt *time.Time
	if org.UpdatedAt != nil {
		updatedAt = &org.UpdatedAt.Time
	}

	return &resource.Resource{
		ID:         fmt.Sprintf("%d", safeInt64(org.ID)),
		Type:       resource.TypeGitHubOrganization,
		Name:       safeString(org.Login),
		Provider:   "github",
		Account:    safeString(org.Login),
		Properties: properties,
		RawData:    org,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

// convertRepositoryToResource converts a GitHub repository to a Resource
func (p *Provider) convertRepositoryToResource(repo *github.Repository, org string) *resource.Resource {
	properties := make(map[string]interface{})

	if repo.Description != nil {
		properties["description"] = *repo.Description
	}
	if repo.Language != nil {
		properties["language"] = *repo.Language
	}
	if repo.StargazersCount != nil {
		properties["stars"] = *repo.StargazersCount
	}
	if repo.ForksCount != nil {
		properties["forks"] = *repo.ForksCount
	}
	if repo.OpenIssuesCount != nil {
		properties["open_issues"] = *repo.OpenIssuesCount
	}
	if repo.Visibility != nil {
		properties["visibility"] = *repo.Visibility
	}
	if repo.Private != nil {
		properties["private"] = *repo.Private
	}
	if repo.DefaultBranch != nil {
		properties["default_branch"] = *repo.DefaultBranch
	}
	if repo.Size != nil {
		properties["size_kb"] = *repo.Size
	}

	var createdAt *time.Time
	if repo.CreatedAt != nil {
		createdAt = &repo.CreatedAt.Time
	}

	var updatedAt *time.Time
	if repo.UpdatedAt != nil {
		updatedAt = &repo.UpdatedAt.Time
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", safeInt64(repo.ID)),
		Type:       resource.TypeGitHubRepository,
		Name:       safeString(repo.Name),
		Provider:   "github",
		Account:    org,
		Properties: properties,
		RawData:    repo,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	// Add organization relationship
	if repo.Owner != nil && repo.Owner.ID != nil {
		res.Relationships = []resource.Relationship{
			{
				Type:       resource.RelationBelongsTo,
				TargetID:   fmt.Sprintf("%d", *repo.Owner.ID),
				TargetType: resource.TypeGitHubOrganization,
			},
		}
	}

	return res
}

// convertTeamToResource converts a GitHub team to a Resource
func (p *Provider) convertTeamToResource(team *github.Team, org string) *resource.Resource {
	properties := make(map[string]interface{})

	if team.Description != nil {
		properties["description"] = *team.Description
	}
	if team.Privacy != nil {
		properties["privacy"] = *team.Privacy
	}
	if team.Permission != nil {
		properties["permission"] = *team.Permission
	}
	if team.MembersCount != nil {
		properties["members_count"] = *team.MembersCount
	}
	if team.ReposCount != nil {
		properties["repos_count"] = *team.ReposCount
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", safeInt64(team.ID)),
		Type:       resource.TypeGitHubTeam,
		Name:       safeString(team.Name),
		Provider:   "github",
		Account:    org,
		Properties: properties,
		RawData:    team,
	}

	// Add organization relationship
	if team.Organization != nil && team.Organization.ID != nil {
		res.Relationships = []resource.Relationship{
			{
				Type:       resource.RelationBelongsTo,
				TargetID:   fmt.Sprintf("%d", *team.Organization.ID),
				TargetType: resource.TypeGitHubOrganization,
			},
		}
	}

	return res
}

// convertUserToResource converts a GitHub user to a Resource
func (p *Provider) convertUserToResource(user *github.User, org string) *resource.Resource {
	properties := make(map[string]interface{})

	if user.Name != nil {
		properties["name"] = *user.Name
	}
	if user.Email != nil {
		properties["email"] = *user.Email
	}
	if user.Company != nil {
		properties["company"] = *user.Company
	}
	if user.Location != nil {
		properties["location"] = *user.Location
	}
	if user.Type != nil {
		properties["type"] = *user.Type
	}
	if user.SiteAdmin != nil {
		properties["site_admin"] = *user.SiteAdmin
	}

	return &resource.Resource{
		ID:         fmt.Sprintf("%d", safeInt64(user.ID)),
		Type:       resource.TypeGitHubUser,
		Name:       safeString(user.Login),
		Provider:   "github",
		Account:    org,
		Properties: properties,
		RawData:    user,
	}
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeInt64 safely dereferences an int64 pointer
func safeInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
