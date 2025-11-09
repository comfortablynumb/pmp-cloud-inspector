package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectIAMUsers collects all IAM users
func (p *Provider) collectIAMUsers(ctx context.Context, collection *resource.Collection) error {
	paginator := iam.NewListUsersPaginator(p.iamClient, &iam.ListUsersInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list IAM users: %w", err)
		}

		for _, user := range output.Users {
			res := p.convertIAMUserToResource(&user)
			collection.Add(res)
		}
	}

	return nil
}

// collectIAMRoles collects all IAM roles
func (p *Provider) collectIAMRoles(ctx context.Context, collection *resource.Collection) error {
	paginator := iam.NewListRolesPaginator(p.iamClient, &iam.ListRolesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list IAM roles: %w", err)
		}

		for _, role := range output.Roles {
			res := p.convertIAMRoleToResource(&role)
			collection.Add(res)
		}
	}

	return nil
}

// collectAccounts collects account information
func (p *Provider) collectAccounts(ctx context.Context, collection *resource.Collection) error {
	for _, accountID := range p.accounts {
		res := &resource.Resource{
			ID:       accountID,
			Type:     resource.TypeAWSAccount,
			Name:     accountID,
			Provider: "aws",
			Account:  accountID,
			Properties: map[string]interface{}{
				"account_id": accountID,
			},
		}
		collection.Add(res)
	}

	return nil
}

// convertIAMUserToResource converts an IAM user to a Resource
func (p *Provider) convertIAMUserToResource(user *iamTypes.User) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{}
	if user.UserId != nil {
		properties["user_id"] = *user.UserId
	}
	if user.Path != nil {
		properties["path"] = *user.Path
	}
	if user.CreateDate != nil {
		properties["create_date"] = user.CreateDate.Format(time.RFC3339)
	}
	if user.PasswordLastUsed != nil {
		properties["password_last_used"] = user.PasswordLastUsed.Format(time.RFC3339)
	}

	var tags map[string]string
	if len(user.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range user.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	var createdAt *time.Time
	if user.CreateDate != nil {
		createdAt = user.CreateDate
	}

	res := &resource.Resource{
		ID:         safeString(user.Arn),
		Type:       resource.TypeAWSIAMUser,
		Name:       safeString(user.UserName),
		Provider:   "aws",
		Account:    account,
		ARN:        safeString(user.Arn),
		Tags:       tags,
		Properties: properties,
		RawData:    user,
		CreatedAt:  createdAt,
	}

	return res
}

// convertIAMRoleToResource converts an IAM role to a Resource
func (p *Provider) convertIAMRoleToResource(role *iamTypes.Role) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{}
	if role.RoleId != nil {
		properties["role_id"] = *role.RoleId
	}
	if role.Path != nil {
		properties["path"] = *role.Path
	}
	if role.CreateDate != nil {
		properties["create_date"] = role.CreateDate.Format(time.RFC3339)
	}
	if role.MaxSessionDuration != nil {
		properties["max_session_duration"] = *role.MaxSessionDuration
	}
	if role.Description != nil {
		properties["description"] = *role.Description
	}

	var tags map[string]string
	if len(role.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range role.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	var createdAt *time.Time
	if role.CreateDate != nil {
		createdAt = role.CreateDate
	}

	res := &resource.Resource{
		ID:         safeString(role.Arn),
		Type:       resource.TypeAWSIAMRole,
		Name:       safeString(role.RoleName),
		Provider:   "aws",
		Account:    account,
		ARN:        safeString(role.Arn),
		Tags:       tags,
		Properties: properties,
		RawData:    role,
		CreatedAt:  createdAt,
	}

	return res
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
