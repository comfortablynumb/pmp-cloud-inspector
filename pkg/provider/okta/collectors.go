//go:build okta
// +build okta

package okta

import (
	"context"
	"fmt"
	"os"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectUsers collects all Okta users
func (p *Provider) collectUsers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Okta users...\n")

	users, _, err := p.client.User.ListUsers(ctx, &query.Params{})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	userCount := 0
	for _, user := range users {
		properties := map[string]interface{}{
			"status":                user.Status,
			"created":               user.Created,
			"activated":             user.Activated,
			"statusChanged":         user.StatusChanged,
			"lastLogin":             user.LastLogin,
			"lastUpdated":           user.LastUpdated,
			"passwordChanged":       user.PasswordChanged,
			"transitioningToStatus": user.TransitioningToStatus,
		}

		if user.Profile != nil {
			properties["profile"] = *user.Profile
			// Extract email from profile if available
			if email, ok := (*user.Profile)["email"]; ok {
				properties["email"] = email
			}
			if firstName, ok := (*user.Profile)["firstName"]; ok {
				properties["firstName"] = firstName
			}
			if lastName, ok := (*user.Profile)["lastName"]; ok {
				properties["lastName"] = lastName
			}
		}

		name := user.Id
		if user.Profile != nil {
			if email, ok := (*user.Profile)["email"]; ok {
				if emailStr, ok := email.(string); ok {
					name = emailStr
				}
			}
		}

		res := &resource.Resource{
			ID:         user.Id,
			Type:       resource.TypeOktaUser,
			Name:       name,
			Provider:   "okta",
			Properties: properties,
			RawData:    user,
			CreatedAt:  user.Created,
			UpdatedAt:  user.LastUpdated,
		}

		collection.Add(res)
		userCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d users\n", userCount)
	return nil
}

// collectGroups collects all Okta groups
func (p *Provider) collectGroups(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Okta groups...\n")

	groups, _, err := p.client.Group.ListGroups(ctx, &query.Params{})
	if err != nil {
		return fmt.Errorf("failed to list groups: %w", err)
	}

	groupCount := 0
	for _, group := range groups {
		properties := map[string]interface{}{
			"type":                  group.Type,
			"created":               group.Created,
			"lastUpdated":           group.LastUpdated,
			"lastMembershipUpdated": group.LastMembershipUpdated,
		}

		if group.Profile != nil {
			properties["profile"] = group.Profile
			if group.Profile.Name != "" {
				properties["name"] = group.Profile.Name
			}
			if group.Profile.Description != "" {
				properties["description"] = group.Profile.Description
			}
		}

		name := group.Id
		if group.Profile != nil && group.Profile.Name != "" {
			name = group.Profile.Name
		}

		res := &resource.Resource{
			ID:         group.Id,
			Type:       resource.TypeOktaGroup,
			Name:       name,
			Provider:   "okta",
			Properties: properties,
			RawData:    group,
			CreatedAt:  group.Created,
			UpdatedAt:  group.LastUpdated,
		}

		collection.Add(res)
		groupCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d groups\n", groupCount)
	return nil
}

// collectApplications collects all Okta applications
func (p *Provider) collectApplications(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Okta applications...\n")

	apps, _, err := p.client.Application.ListApplications(ctx, &query.Params{})
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	appCount := 0
	for _, app := range apps {
		// Type assert to *Application to access fields
		appDetails, ok := app.(*okta.Application)
		if !ok {
			fmt.Fprintf(os.Stderr, "    Warning: could not type assert application\n")
			continue
		}

		properties := map[string]interface{}{
			"status":      appDetails.Status,
			"created":     appDetails.Created,
			"lastUpdated": appDetails.LastUpdated,
			"signOnMode":  appDetails.SignOnMode,
			"name":        appDetails.Name,
		}

		if appDetails.Label != "" {
			properties["label"] = appDetails.Label
		}

		if appDetails.Settings != nil {
			properties["settings"] = appDetails.Settings
		}

		if appDetails.Credentials != nil {
			// Don't include sensitive credential data, just metadata about credentials
			if appDetails.Credentials.Signing != nil {
				properties["credentialsSigning"] = appDetails.Credentials.Signing
			}
			if appDetails.Credentials.UserNameTemplate != nil {
				properties["credentialsUserNameTemplate"] = appDetails.Credentials.UserNameTemplate
			}
		}

		if appDetails.Profile != nil {
			properties["profile"] = appDetails.Profile
		}

		name := appDetails.Id
		if appDetails.Label != "" {
			name = appDetails.Label
		}

		res := &resource.Resource{
			ID:         appDetails.Id,
			Type:       resource.TypeOktaApplication,
			Name:       name,
			Provider:   "okta",
			Properties: properties,
			RawData:    appDetails,
			CreatedAt:  appDetails.Created,
			UpdatedAt:  appDetails.LastUpdated,
		}

		collection.Add(res)
		appCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d applications\n", appCount)
	return nil
}

// collectAuthorizationServers collects all Okta authorization servers
func (p *Provider) collectAuthorizationServers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Okta authorization servers...\n")

	authServers, _, err := p.client.AuthorizationServer.ListAuthorizationServers(ctx, &query.Params{})
	if err != nil {
		return fmt.Errorf("failed to list authorization servers: %w", err)
	}

	serverCount := 0
	for _, server := range authServers {
		properties := map[string]interface{}{
			"status":      server.Status,
			"created":     server.Created,
			"lastUpdated": server.LastUpdated,
			"issuer":      server.Issuer,
			"audiences":   server.Audiences,
		}

		if server.Name != "" {
			properties["name"] = server.Name
		}

		if server.Description != "" {
			properties["description"] = server.Description
		}

		if server.Credentials != nil && server.Credentials.Signing != nil {
			properties["credentialsSigning"] = server.Credentials.Signing
		}

		name := server.Id
		if server.Name != "" {
			name = server.Name
		}

		res := &resource.Resource{
			ID:         server.Id,
			Type:       resource.TypeOktaAuthorizationServer,
			Name:       name,
			Provider:   "okta",
			Properties: properties,
			RawData:    server,
			CreatedAt:  server.Created,
			UpdatedAt:  server.LastUpdated,
		}

		collection.Add(res)
		serverCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d authorization servers\n", serverCount)
	return nil
}
