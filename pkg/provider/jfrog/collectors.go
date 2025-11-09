package jfrog

import (
	"context"
	"fmt"
	"os"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Repository represents a JFrog repository
type Repository struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Description string `json:"description"`
	URL         string `json:"url"`
	PackageType string `json:"packageType"`
}

// User represents a JFrog user
type User struct {
	Name                     string   `json:"name"`
	Email                    string   `json:"email"`
	Admin                    bool     `json:"admin"`
	ProfileUpdatable         bool     `json:"profileUpdatable"`
	InternalPasswordDisabled bool     `json:"internalPasswordDisabled"`
	Groups                   []string `json:"groups"`
}

// Group represents a JFrog group
type Group struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	AutoJoin        bool   `json:"autoJoin"`
	AdminPrivileges bool   `json:"adminPrivileges"`
}

// Permission represents a JFrog permission target
type Permission struct {
	Name         string                 `json:"name"`
	Repositories []string               `json:"repositories"`
	Principals   map[string]interface{} `json:"principals"`
}

// collectRepositories collects all repositories
func (p *Provider) collectRepositories(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting JFrog repositories...\n")

	resp, err := p.doRequest("GET", "repositories")
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []Repository
	if err := parseResponse(resp, &repos); err != nil {
		return fmt.Errorf("failed to parse repositories: %w", err)
	}

	count := 0
	for _, repo := range repos {
		res := p.convertRepositoryToResource(&repo)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found repository: %s (%s)\n", repo.Key, repo.Type)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d JFrog repositories\n", count)
	return nil
}

// collectUsers collects all users
func (p *Provider) collectUsers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting JFrog users...\n")

	resp, err := p.doRequest("GET", "security/users")
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	var users []User
	if err := parseResponse(resp, &users); err != nil {
		return fmt.Errorf("failed to parse users: %w", err)
	}

	count := 0
	for _, user := range users {
		res := p.convertUserToResource(&user)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found user: %s\n", user.Name)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d JFrog users\n", count)
	return nil
}

// collectGroups collects all groups
func (p *Provider) collectGroups(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting JFrog groups...\n")

	resp, err := p.doRequest("GET", "security/groups")
	if err != nil {
		return fmt.Errorf("failed to list groups: %w", err)
	}

	var groups []Group
	if err := parseResponse(resp, &groups); err != nil {
		return fmt.Errorf("failed to parse groups: %w", err)
	}

	count := 0
	for _, group := range groups {
		res := p.convertGroupToResource(&group)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found group: %s\n", group.Name)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d JFrog groups\n", count)
	return nil
}

// collectPermissions collects all permission targets
func (p *Provider) collectPermissions(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting JFrog permissions...\n")

	resp, err := p.doRequest("GET", "security/permissions")
	if err != nil {
		return fmt.Errorf("failed to list permissions: %w", err)
	}

	var permissions []Permission
	if err := parseResponse(resp, &permissions); err != nil {
		return fmt.Errorf("failed to parse permissions: %w", err)
	}

	count := 0
	for _, perm := range permissions {
		res := p.convertPermissionToResource(&perm)
		collection.Add(res)
		count++
		fmt.Fprintf(os.Stderr, "    Found permission: %s\n", perm.Name)
	}

	fmt.Fprintf(os.Stderr, "  Collected %d JFrog permissions\n", count)
	return nil
}

// convertRepositoryToResource converts a JFrog repository to a Resource
func (p *Provider) convertRepositoryToResource(repo *Repository) *resource.Resource {
	properties := map[string]interface{}{
		"type":         repo.Type,
		"description":  repo.Description,
		"url":          repo.URL,
		"package_type": repo.PackageType,
	}

	return &resource.Resource{
		ID:         repo.Key,
		Type:       resource.TypeJFrogRepository,
		Name:       repo.Key,
		Provider:   "jfrog",
		Properties: properties,
		RawData:    repo,
	}
}

// convertUserToResource converts a JFrog user to a Resource
func (p *Provider) convertUserToResource(user *User) *resource.Resource {
	properties := map[string]interface{}{
		"email":                      user.Email,
		"admin":                      user.Admin,
		"profile_updatable":          user.ProfileUpdatable,
		"internal_password_disabled": user.InternalPasswordDisabled,
		"groups":                     user.Groups,
	}

	res := &resource.Resource{
		ID:         user.Name,
		Type:       resource.TypeJFrogUser,
		Name:       user.Name,
		Provider:   "jfrog",
		Properties: properties,
		RawData:    user,
	}

	// Add group relationships
	for _, groupName := range user.Groups {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   groupName,
			TargetType: resource.TypeJFrogGroup,
		})
	}

	return res
}

// convertGroupToResource converts a JFrog group to a Resource
func (p *Provider) convertGroupToResource(group *Group) *resource.Resource {
	properties := map[string]interface{}{
		"description":      group.Description,
		"auto_join":        group.AutoJoin,
		"admin_privileges": group.AdminPrivileges,
	}

	return &resource.Resource{
		ID:         group.Name,
		Type:       resource.TypeJFrogGroup,
		Name:       group.Name,
		Provider:   "jfrog",
		Properties: properties,
		RawData:    group,
	}
}

// convertPermissionToResource converts a JFrog permission to a Resource
func (p *Provider) convertPermissionToResource(perm *Permission) *resource.Resource {
	properties := map[string]interface{}{
		"repositories": perm.Repositories,
		"principals":   perm.Principals,
	}

	res := &resource.Resource{
		ID:         perm.Name,
		Type:       resource.TypeJFrogPermission,
		Name:       perm.Name,
		Provider:   "jfrog",
		Properties: properties,
		RawData:    perm,
	}

	// Add repository relationships
	for _, repoKey := range perm.Repositories {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationReferences,
			TargetID:   repoKey,
			TargetType: resource.TypeJFrogRepository,
		})
	}

	return res
}
