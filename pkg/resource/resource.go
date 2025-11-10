package resource

import (
	"encoding/json"
	"time"
)

// ResourceType represents the type of cloud resource
type ResourceType string

const (
	// AWS Resource Types
	TypeAWSIAMUser       ResourceType = "aws:iam:user"
	TypeAWSIAMRole       ResourceType = "aws:iam:role"
	TypeAWSAccount       ResourceType = "aws:account"
	TypeAWSVPC           ResourceType = "aws:ec2:vpc"
	TypeAWSSubnet        ResourceType = "aws:ec2:subnet"
	TypeAWSSecurityGroup ResourceType = "aws:ec2:security-group"
	TypeAWSEC2Instance   ResourceType = "aws:ec2:instance"
	TypeAWSECR           ResourceType = "aws:ecr:repository"
	TypeAWSEKSCluster    ResourceType = "aws:eks:cluster"
	TypeAWSELB           ResourceType = "aws:elb:classic"
	TypeAWSALB           ResourceType = "aws:elb:application"
	TypeAWSNLB           ResourceType = "aws:elb:network"
	TypeAWSLambda        ResourceType = "aws:lambda:function"
	TypeAWSAPIGateway    ResourceType = "aws:apigateway:api"
	TypeAWSCloudFront    ResourceType = "aws:cloudfront:distribution"
	TypeAWSMemoryDB      ResourceType = "aws:memorydb:cluster"
	TypeAWSElastiCache   ResourceType = "aws:elasticache:cluster"
	TypeAWSSecret        ResourceType = "aws:secretsmanager:secret"
	TypeAWSSNSTopic      ResourceType = "aws:sns:topic"
	TypeAWSSQSQueue      ResourceType = "aws:sqs:queue"
	TypeAWSDynamoDBTable ResourceType = "aws:dynamodb:table"

	// GitHub Resource Types
	TypeGitHubOrganization ResourceType = "github:organization"
	TypeGitHubRepository   ResourceType = "github:repository"
	TypeGitHubTeam         ResourceType = "github:team"
	TypeGitHubUser         ResourceType = "github:user"

	// GitLab Resource Types
	TypeGitLabProject ResourceType = "gitlab:project"
	TypeGitLabGroup   ResourceType = "gitlab:group"
	TypeGitLabUser    ResourceType = "gitlab:user"

	// JFrog Resource Types
	TypeJFrogRepository ResourceType = "jfrog:repository"
	TypeJFrogUser       ResourceType = "jfrog:user"
	TypeJFrogGroup      ResourceType = "jfrog:group"
	TypeJFrogPermission ResourceType = "jfrog:permission"

	// GCP Resource Types
	TypeGCPProject         ResourceType = "gcp:project"
	TypeGCPComputeInstance ResourceType = "gcp:compute:instance"
	TypeGCPVPC             ResourceType = "gcp:compute:network"
	TypeGCPSubnet          ResourceType = "gcp:compute:subnetwork"
	TypeGCPStorageBucket   ResourceType = "gcp:storage:bucket"
	TypeGCPCloudFunction   ResourceType = "gcp:cloudfunctions:function"
	TypeGCPCloudRun        ResourceType = "gcp:run:service"

	// Okta Resource Types
	TypeOktaUser                ResourceType = "okta:user"
	TypeOktaGroup               ResourceType = "okta:group"
	TypeOktaApplication         ResourceType = "okta:application"
	TypeOktaAuthorizationServer ResourceType = "okta:authorizationserver"

	// Auth0 Resource Types
	TypeAuth0User           ResourceType = "auth0:user"
	TypeAuth0Role           ResourceType = "auth0:role"
	TypeAuth0Client         ResourceType = "auth0:client"
	TypeAuth0ResourceServer ResourceType = "auth0:resourceserver"
	TypeAuth0Connection     ResourceType = "auth0:connection"

	// Azure Resource Types
	TypeAzureResourceGroup  ResourceType = "azure:resourcegroup"
	TypeAzureVM             ResourceType = "azure:compute:vm"
	TypeAzureVNet           ResourceType = "azure:network:vnet"
	TypeAzureSubnet         ResourceType = "azure:network:subnet"
	TypeAzureStorageAccount ResourceType = "azure:storage:account"
	TypeAzureAppService     ResourceType = "azure:web:appservice"
	TypeAzureSQLDatabase    ResourceType = "azure:sql:database"
	TypeAzureKeyVault       ResourceType = "azure:keyvault:vault"
)

// Resource represents a cloud resource
type Resource struct {
	ID            string                 `json:"id"`
	Type          ResourceType           `json:"type"`
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider"`
	Account       string                 `json:"account,omitempty"`
	Region        string                 `json:"region,omitempty"`
	ARN           string                 `json:"arn,omitempty"` // AWS specific, but can be used for unique identifiers
	Tags          map[string]string      `json:"tags,omitempty"`
	Properties    map[string]interface{} `json:"properties"`
	RawData       interface{}            `json:"raw_data,omitempty"`
	Relationships []Relationship         `json:"relationships,omitempty"`
	Cost          *ResourceCost          `json:"cost,omitempty"`
	CreatedAt     *time.Time             `json:"created_at,omitempty"`
	UpdatedAt     *time.Time             `json:"updated_at,omitempty"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	MonthlyEstimate float64            `json:"monthly_estimate"`    // Estimated monthly cost
	Currency        string             `json:"currency"`            // Currency code (USD, EUR, etc.)
	Breakdown       map[string]float64 `json:"breakdown,omitempty"` // Cost breakdown by component
	LastUpdated     time.Time          `json:"last_updated"`        // When cost was last calculated
}

// Relationship represents a connection between resources
type Relationship struct {
	Type       RelationType           `json:"type"`
	TargetID   string                 `json:"target_id"`
	TargetType ResourceType           `json:"target_type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// RelationType defines types of relationships between resources
type RelationType string

const (
	RelationContains   RelationType = "contains"    // e.g., VPC contains Subnets
	RelationBelongsTo  RelationType = "belongs_to"  // e.g., Subnet belongs to VPC
	RelationAttachedTo RelationType = "attached_to" // e.g., SecurityGroup attached to Instance
	RelationAssumes    RelationType = "assumes"     // e.g., Service assumes Role
	RelationHasAccess  RelationType = "has_access"  // e.g., User has access to Resource
	RelationReferences RelationType = "references"  // Generic reference
	RelationDependsOn  RelationType = "depends_on"  // Dependency relationship
)

// Collection holds all discovered resources
type Collection struct {
	Resources []*Resource          `json:"resources"`
	Metadata  CollectionMetadata   `json:"metadata"`
	index     map[string]*Resource // internal index for quick lookups
}

// CollectionMetadata provides information about the collection
type CollectionMetadata struct {
	Timestamp       time.Time                       `json:"timestamp"`
	TotalCount      int                             `json:"total_count"`
	ByType          map[ResourceType]int            `json:"by_type"`
	ByProvider      map[string]int                  `json:"by_provider"`
	ByAccount       map[string]int                  `json:"by_account,omitempty"`
	ByRegion        map[string]int                  `json:"by_region,omitempty"`
	ByTypeAndRegion map[string]map[ResourceType]int `json:"by_type_and_region,omitempty"`
	TotalCost       *CostSummary                    `json:"total_cost,omitempty"`
}

// CostSummary provides cost aggregations for the collection
type CostSummary struct {
	Total      float64            `json:"total"`                 // Total monthly cost estimate
	Currency   string             `json:"currency"`              // Currency code
	ByProvider map[string]float64 `json:"by_provider,omitempty"` // Cost breakdown by provider
	ByRegion   map[string]float64 `json:"by_region,omitempty"`   // Cost breakdown by region
	ByType     map[string]float64 `json:"by_type,omitempty"`     // Cost breakdown by resource type
	ByTag      map[string]float64 `json:"by_tag,omitempty"`      // Cost breakdown by tag values
}

// NewCollection creates a new resource collection
func NewCollection() *Collection {
	return &Collection{
		Resources: make([]*Resource, 0),
		index:     make(map[string]*Resource),
		Metadata: CollectionMetadata{
			Timestamp:       time.Now(),
			ByType:          make(map[ResourceType]int),
			ByProvider:      make(map[string]int),
			ByAccount:       make(map[string]int),
			ByRegion:        make(map[string]int),
			ByTypeAndRegion: make(map[string]map[ResourceType]int),
		},
	}
}

// Add adds a resource to the collection
func (c *Collection) Add(resource *Resource) {
	c.Resources = append(c.Resources, resource)
	c.index[resource.ID] = resource

	// Update metadata
	c.Metadata.TotalCount++
	c.Metadata.ByType[resource.Type]++
	c.Metadata.ByProvider[resource.Provider]++

	if resource.Account != "" {
		c.Metadata.ByAccount[resource.Account]++
	}

	if resource.Region != "" {
		c.Metadata.ByRegion[resource.Region]++

		// Update by type and region
		if c.Metadata.ByTypeAndRegion[resource.Region] == nil {
			c.Metadata.ByTypeAndRegion[resource.Region] = make(map[ResourceType]int)
		}
		c.Metadata.ByTypeAndRegion[resource.Region][resource.Type]++
	}

	// Update cost metadata
	if resource.Cost != nil && resource.Cost.MonthlyEstimate > 0 {
		c.updateCostMetadata(resource)
	}
}

// updateCostMetadata updates cost aggregations in metadata
func (c *Collection) updateCostMetadata(resource *Resource) {
	// Initialize cost summary if needed
	if c.Metadata.TotalCost == nil {
		c.Metadata.TotalCost = &CostSummary{
			Currency:   resource.Cost.Currency,
			ByProvider: make(map[string]float64),
			ByRegion:   make(map[string]float64),
			ByType:     make(map[string]float64),
			ByTag:      make(map[string]float64),
		}
	}

	cost := resource.Cost.MonthlyEstimate

	// Update total
	c.Metadata.TotalCost.Total += cost

	// Update by provider
	c.Metadata.TotalCost.ByProvider[resource.Provider] += cost

	// Update by region
	if resource.Region != "" {
		c.Metadata.TotalCost.ByRegion[resource.Region] += cost
	}

	// Update by type
	c.Metadata.TotalCost.ByType[string(resource.Type)] += cost

	// Update by tags (sum costs for each tag key-value pair)
	for key, value := range resource.Tags {
		tagKey := key + "=" + value
		c.Metadata.TotalCost.ByTag[tagKey] += cost
	}
}

// Get retrieves a resource by ID
func (c *Collection) Get(id string) *Resource {
	return c.index[id]
}

// Filter returns resources matching the given filter function
func (c *Collection) Filter(fn func(*Resource) bool) []*Resource {
	var filtered []*Resource
	for _, r := range c.Resources {
		if fn(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// MarshalJSON implements custom JSON marshaling
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Resources []*Resource        `json:"resources"`
		Metadata  CollectionMetadata `json:"metadata"`
	}{
		Resources: c.Resources,
		Metadata:  c.Metadata,
	})
}
