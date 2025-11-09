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

	// GitHub Resource Types
	TypeGitHubOrganization ResourceType = "github:organization"
	TypeGitHubRepository   ResourceType = "github:repository"
	TypeGitHubTeam         ResourceType = "github:team"
	TypeGitHubUser         ResourceType = "github:user"

	// Future providers can add their types here
	// TypeGCPProject ResourceType = "gcp:project"
	// TypeOktaUser   ResourceType = "okta:user"
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
	CreatedAt     *time.Time             `json:"created_at,omitempty"`
	UpdatedAt     *time.Time             `json:"updated_at,omitempty"`
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
