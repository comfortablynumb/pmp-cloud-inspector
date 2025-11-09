//go:build gitlab
// +build gitlab

package gitlab

import (
	"context"
	"fmt"
	"os"

	"github.com/xanzy/go-gitlab"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectGroups collects all groups
func (p *Provider) collectGroups(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GitLab groups...\n")

	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	count := 0
	for {
		groups, resp, err := p.client.Groups.ListGroups(opt)
		if err != nil {
			return fmt.Errorf("failed to list groups: %w", err)
		}

		for _, group := range groups {
			res := p.convertGroupToResource(group)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found group: %s (%s)\n", group.Name, group.FullPath)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d GitLab groups\n", count)
	return nil
}

// collectProjects collects all projects
func (p *Provider) collectProjects(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GitLab projects...\n")

	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	// If groups are specified, collect projects for those groups
	if len(p.groups) > 0 {
		count := 0
		for _, groupPath := range p.groups {
			groupOpt := &gitlab.ListGroupProjectsOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 100,
					Page:    1,
				},
			}

			for {
				projects, resp, err := p.client.Groups.ListGroupProjects(groupPath, groupOpt)
				if err != nil {
					return fmt.Errorf("failed to list projects for group %s: %w", groupPath, err)
				}

				for _, project := range projects {
					res := p.convertProjectToResource(project)
					collection.Add(res)
					count++
					fmt.Fprintf(os.Stderr, "    Found project: %s (%s)\n", project.Name, project.PathWithNamespace)
				}

				if resp.NextPage == 0 {
					break
				}
				groupOpt.Page = resp.NextPage
			}
		}
		fmt.Fprintf(os.Stderr, "  Collected %d GitLab projects\n", count)
		return nil
	}

	// Otherwise collect all accessible projects
	count := 0
	for {
		projects, resp, err := p.client.Projects.ListProjects(opt)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		for _, project := range projects {
			res := p.convertProjectToResource(project)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found project: %s (%s)\n", project.Name, project.PathWithNamespace)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d GitLab projects\n", count)
	return nil
}

// collectUsers collects all users
func (p *Provider) collectUsers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting GitLab users...\n")

	opt := &gitlab.ListUsersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	count := 0
	for {
		users, resp, err := p.client.Users.ListUsers(opt)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		for _, user := range users {
			res := p.convertUserToResource(user)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found user: %s (%s)\n", user.Name, user.Username)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	fmt.Fprintf(os.Stderr, "  Collected %d GitLab users\n", count)
	return nil
}

// convertGroupToResource converts a GitLab group to a Resource
func (p *Provider) convertGroupToResource(group *gitlab.Group) *resource.Resource {
	properties := map[string]interface{}{
		"full_path":   group.FullPath,
		"visibility":  group.Visibility,
		"description": group.Description,
	}

	if group.ParentID != 0 {
		properties["parent_id"] = group.ParentID
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", group.ID),
		Type:       resource.TypeGitLabGroup,
		Name:       group.Name,
		Provider:   "gitlab",
		Properties: properties,
		RawData:    group,
	}

	if group.CreatedAt != nil {
		res.CreatedAt = group.CreatedAt
	}

	return res
}

// convertProjectToResource converts a GitLab project to a Resource
func (p *Provider) convertProjectToResource(project *gitlab.Project) *resource.Resource {
	properties := map[string]interface{}{
		"path_with_namespace": project.PathWithNamespace,
		"visibility":          project.Visibility,
		"description":         project.Description,
		"default_branch":      project.DefaultBranch,
		"archived":            project.Archived,
	}

	if project.WebURL != "" {
		properties["web_url"] = project.WebURL
	}
	if project.SSHURLToRepo != "" {
		properties["ssh_url"] = project.SSHURLToRepo
	}
	if project.HTTPURLToRepo != "" {
		properties["http_url"] = project.HTTPURLToRepo
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", project.ID),
		Type:       resource.TypeGitLabProject,
		Name:       project.Name,
		Provider:   "gitlab",
		Properties: properties,
		RawData:    project,
	}

	if project.CreatedAt != nil {
		res.CreatedAt = project.CreatedAt
	}
	if project.LastActivityAt != nil {
		res.UpdatedAt = project.LastActivityAt
	}

	// Add namespace relationship
	if project.Namespace != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   fmt.Sprintf("%d", project.Namespace.ID),
			TargetType: resource.TypeGitLabGroup,
		})
	}

	return res
}

// convertUserToResource converts a GitLab user to a Resource
func (p *Provider) convertUserToResource(user *gitlab.User) *resource.Resource {
	properties := map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"state":    user.State,
	}

	if user.WebURL != "" {
		properties["web_url"] = user.WebURL
	}
	if user.AvatarURL != "" {
		properties["avatar_url"] = user.AvatarURL
	}

	res := &resource.Resource{
		ID:         fmt.Sprintf("%d", user.ID),
		Type:       resource.TypeGitLabUser,
		Name:       user.Name,
		Provider:   "gitlab",
		Properties: properties,
		RawData:    user,
	}

	if user.CreatedAt != nil {
		res.CreatedAt = user.CreatedAt
	}

	return res
}

// discoverProjectRelationships discovers relationships for projects
func (p *Provider) discoverProjectRelationships(res *resource.Resource, collection *resource.Collection) {
	// Relationships are already added during conversion
}
