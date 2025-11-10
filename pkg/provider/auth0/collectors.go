//go:build auth0
// +build auth0

package auth0

import (
	"context"
	"fmt"
	"os"

	"github.com/auth0/go-auth0/management"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectUsers collects all Auth0 users
func (p *Provider) collectUsers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Auth0 users...\n")

	var page int
	var perPage int = 100
	var allUsers []*management.User

	for {
		users, err := p.client.User.List(
			ctx,
			management.Page(page),
			management.PerPage(perPage),
		)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		if users == nil || len(users.Users) == 0 {
			break
		}

		allUsers = append(allUsers, users.Users...)

		if !users.HasNext() {
			break
		}
		page++
	}

	for _, user := range allUsers {
		if user.ID == nil {
			continue
		}

		properties := map[string]interface{}{
			"user_id":        user.GetID(),
			"email":          user.GetEmail(),
			"email_verified": user.GetEmailVerified(),
			"username":       user.GetUsername(),
			"phone_number":   user.GetPhoneNumber(),
			"created_at":     user.GetCreatedAt(),
			"updated_at":     user.GetUpdatedAt(),
			"last_login":     user.GetLastLogin(),
			"logins_count":   user.GetLoginsCount(),
			"blocked":        user.GetBlocked(),
		}

		if user.Name != nil {
			properties["name"] = *user.Name
		}

		if user.UserMetadata != nil {
			properties["user_metadata"] = user.UserMetadata
		}

		if user.AppMetadata != nil {
			properties["app_metadata"] = user.AppMetadata
		}

		name := user.GetEmail()
		if name == "" {
			name = user.GetID()
		}

		res := &resource.Resource{
			ID:         user.GetID(),
			Type:       resource.TypeAuth0User,
			Name:       name,
			Provider:   "auth0",
			Properties: properties,
			RawData:    user,
		}

		collection.Add(res)
	}

	fmt.Fprintf(os.Stderr, "    Found %d users\n", len(allUsers))
	return nil
}

// collectRoles collects all Auth0 roles
func (p *Provider) collectRoles(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Auth0 roles...\n")

	roles, err := p.client.Role.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	if roles == nil || roles.Roles == nil {
		fmt.Fprintf(os.Stderr, "    Found 0 roles\n")
		return nil
	}

	for _, role := range roles.Roles {
		if role.ID == nil {
			continue
		}

		properties := map[string]interface{}{
			"name":        role.GetName(),
			"description": role.GetDescription(),
		}

		res := &resource.Resource{
			ID:         role.GetID(),
			Type:       resource.TypeAuth0Role,
			Name:       role.GetName(),
			Provider:   "auth0",
			Properties: properties,
			RawData:    role,
		}

		collection.Add(res)
	}

	fmt.Fprintf(os.Stderr, "    Found %d roles\n", len(roles.Roles))
	return nil
}

// collectClients collects all Auth0 clients (applications)
func (p *Provider) collectClients(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Auth0 clients...\n")

	clientList, err := p.client.Client.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	if clientList == nil || clientList.Clients == nil {
		fmt.Fprintf(os.Stderr, "    Found 0 clients\n")
		return nil
	}

	clientCount := 0
	for _, client := range clientList.Clients {
		if client.ClientID == nil {
			continue
		}

		properties := map[string]interface{}{
			"client_id":       client.GetClientID(),
			"name":            client.GetName(),
			"description":     client.GetDescription(),
			"app_type":        client.GetAppType(),
			"is_first_party":  client.GetIsFirstParty(),
			"callbacks":       client.Callbacks,
			"allowed_origins": client.AllowedOrigins,
			"web_origins":     client.WebOrigins,
			"grant_types":     client.GrantTypes,
		}

		res := &resource.Resource{
			ID:         client.GetClientID(),
			Type:       resource.TypeAuth0Client,
			Name:       client.GetName(),
			Provider:   "auth0",
			Properties: properties,
			RawData:    client,
		}

		collection.Add(res)
		clientCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d clients\n", clientCount)
	return nil
}

// collectResourceServers collects all Auth0 resource servers (APIs)
func (p *Provider) collectResourceServers(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Auth0 resource servers...\n")

	servers, err := p.client.ResourceServer.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resource servers: %w", err)
	}

	if servers == nil || servers.ResourceServers == nil {
		fmt.Fprintf(os.Stderr, "    Found 0 resource servers\n")
		return nil
	}

	for _, server := range servers.ResourceServers {
		if server.ID == nil {
			continue
		}

		properties := map[string]interface{}{
			"identifier":           server.GetIdentifier(),
			"name":                 server.GetName(),
			"signing_alg":          server.GetSigningAlgorithm(),
			"allow_offline_access": server.GetAllowOfflineAccess(),
			"skip_consent_for_verifiable_first_party_clients": server.GetSkipConsentForVerifiableFirstPartyClients(),
		}

		if server.Scopes != nil {
			properties["scopes"] = server.Scopes
		}

		res := &resource.Resource{
			ID:         server.GetID(),
			Type:       resource.TypeAuth0ResourceServer,
			Name:       server.GetName(),
			Provider:   "auth0",
			Properties: properties,
			RawData:    server,
		}

		collection.Add(res)
	}

	fmt.Fprintf(os.Stderr, "    Found %d resource servers\n", len(servers.ResourceServers))
	return nil
}

// collectConnections collects all Auth0 connections
func (p *Provider) collectConnections(ctx context.Context, collection *resource.Collection) error {
	fmt.Fprintf(os.Stderr, "  Collecting Auth0 connections...\n")

	connections, err := p.client.Connection.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	if connections == nil || connections.Connections == nil {
		fmt.Fprintf(os.Stderr, "    Found 0 connections\n")
		return nil
	}

	for _, conn := range connections.Connections {
		if conn.ID == nil {
			continue
		}

		properties := map[string]interface{}{
			"name":            conn.GetName(),
			"strategy":        conn.GetStrategy(),
			"enabled_clients": conn.EnabledClients,
		}

		res := &resource.Resource{
			ID:         conn.GetID(),
			Type:       resource.TypeAuth0Connection,
			Name:       conn.GetName(),
			Provider:   "auth0",
			Properties: properties,
			RawData:    conn,
		}

		collection.Add(res)
	}

	fmt.Fprintf(os.Stderr, "    Found %d connections\n", len(connections.Connections))
	return nil
}
