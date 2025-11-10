# PMP Cloud Inspector

**PMP Cloud Inspector** is a powerful CLI tool that allows you to inspect your cloud accounts and export found resources to several formats. Part of the Poor Man's Platform ecosystem.

## Features

- **Multi-Provider Support**: Extensible provider architecture supporting AWS (with more providers coming soon)
- **Comprehensive Resource Discovery**: Automatically discovers and catalogs cloud resources
- **Relationship Mapping**: Discovers and tracks relationships between resources
- **Multiple Export Formats**: Export to JSON, YAML, or GraphViz DOT format
- **Web UI**: Beautiful web interface for viewing and exploring resources (Tailwind CSS + jQuery)
- **Flexible Configuration**: YAML-based configuration for fine-grained control
- **Resource Filtering**: Select specific resource types or collect all available resources
- **Search & Filter**: Search and filter resources by provider, type, region, and more in the web UI

## Supported Providers

### AWS
- IAM Users
- IAM Roles
- Accounts
- VPCs
- Subnets
- Security Groups
- EC2 Instances
- ECR Repositories
- EKS Clusters
- Load Balancers (Classic ELB, ALB, NLB)
- Lambda Functions
- API Gateway (REST & HTTP APIs)
- CloudFront Distributions
- MemoryDB Clusters
- ElastiCache Clusters
- Secrets Manager Secrets
- SNS Topics
- SQS Queues
- DynamoDB Tables

### GitHub
- Organizations
- Repositories
- Teams
- Users

### GitLab
- Projects
- Groups
- Users

### JFrog Artifactory
- Repositories
- Users
- Groups
- Permissions

### GCP (Google Cloud Platform)
- Projects
- Compute Instances
- VPC Networks
- Subnetworks
- Storage Buckets
- Cloud Functions
- Cloud Run Services

### Okta
- Users
- Groups
- Applications
- Authorization Servers

### Auth0
- Users
- Roles
- Clients (Applications)
- Resource Servers (APIs)
- Connections

### Azure
- Resource Groups
- Virtual Machines
- Virtual Networks
- Subnets
- Storage Accounts
- App Services
- SQL Databases
- Key Vaults

More providers coming soon!

## Installation

### Quick Install (Linux/macOS)

**First, check the [latest release version](https://github.com/comfortablynumb/pmp-cloud-inspector/releases) and replace `VERSION` below with the actual version (e.g., `v0.2.0`).**

Install with a single command:

```bash
# Linux or macOS (amd64 / arm64) - Change VERSION for the version you want to download. Check: https://github.com/comfortablynumb/pmp-cloud-inspector/releases

VERSION=v0.2.0 curl -sL -o /tmp/pmp-cloud-inspector.tar.gz https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/${VERSION}/pmp-cloud-inspector_${VERSION#v}_$(uname -s)_$(uname -m).tar.gz && tar -xz /tmp/pmp-cloud-inspector.tar.gz && chmod +x /tmp/pmp-cloud-inspector && sudo mv pmp-cloud-inspector /usr/local/bin/
```

Or download, extract, and install manually:
```bash
# Set the version you want to download (check https://github.com/comfortablynumb/pmp-cloud-inspector/releases)
VERSION=v0.2.0

# Download the appropriate tar.gz for your platform
curl -LO https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/${VERSION}/pmp-cloud-inspector_${VERSION#v}_Linux_x86_64.tar.gz

# Extract the archive
tar -xzf pmp-cloud-inspector_${VERSION#v}_Linux_x86_64.tar.gz

# Make executable and move to PATH
chmod +x pmp-cloud-inspector
sudo mv pmp-cloud-inspector /usr/local/bin/
```

### Quick Install (Windows)

Download and extract using PowerShell:

```powershell
# Set the version (check https://github.com/comfortablynumb/pmp-cloud-inspector/releases)
$VERSION = "v0.2.0"
$VERSION_NUMBER = $VERSION.TrimStart("v")

# Download the release
Invoke-WebRequest -Uri "https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/$VERSION/pmp-cloud-inspector_${VERSION_NUMBER}_Windows_x86_64.zip" -OutFile "pmp-cloud-inspector.zip"

# Extract the archive
Expand-Archive -Path "pmp-cloud-inspector.zip" -DestinationPath "." -Force

# Move to a directory in your PATH (e.g., C:\Program Files\pmp-cloud-inspector\)
# Or run directly from the current directory
.\pmp-cloud-inspector.exe --help
```

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/comfortablynumb/pmp-cloud-inspector/releases) page.

**Available formats (replace `VERSION` with actual version like `0.2.0`):**
- **Linux**: `pmp-cloud-inspector_VERSION_Linux_x86_64.tar.gz`, `pmp-cloud-inspector_VERSION_Linux_arm64.tar.gz`
- **macOS**: `pmp-cloud-inspector_VERSION_Darwin_x86_64.tar.gz`, `pmp-cloud-inspector_VERSION_Darwin_arm64.tar.gz`
- **Windows**: `pmp-cloud-inspector_VERSION_Windows_x86_64.zip`

**Example URLs for version v0.2.0:**
- Linux amd64: `https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/v0.2.0/pmp-cloud-inspector_0.2.0_Linux_x86_64.tar.gz`
- macOS arm64: `https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/v0.2.0/pmp-cloud-inspector_0.2.0_Darwin_arm64.tar.gz`
- Windows: `https://github.com/comfortablynumb/pmp-cloud-inspector/releases/download/v0.2.0/pmp-cloud-inspector_0.2.0_Windows_x86_64.zip`

All binaries are built with support for all providers (AWS, GitHub, GitLab, JFrog, GCP, Okta, Auth0, Azure).

### From Source

Requirements:
- Go 1.24.7 or higher
- Provider credentials configured (see [Provider Authentication](#provider-authentication))

```bash
git clone https://github.com/comfortablynumb/pmp-cloud-inspector.git
cd pmp-cloud-inspector

# Basic build (AWS and GitHub providers only)
go build -o pmp-cloud-inspector ./cmd/inspector

# Build with all providers (requires downloading dependencies)
go mod tidy
go build -tags "gitlab jfrog gcp okta auth0 azure" -o pmp-cloud-inspector ./cmd/inspector

# Or build with specific providers only
go build -tags "gitlab" -o pmp-cloud-inspector ./cmd/inspector  # GitLab only
go build -tags "gcp" -o pmp-cloud-inspector ./cmd/inspector     # GCP only
go build -tags "okta" -o pmp-cloud-inspector ./cmd/inspector    # Okta only
go build -tags "azure" -o pmp-cloud-inspector ./cmd/inspector   # Azure only
```

## Quick Start

1. Create a configuration file (see `examples/config.yaml`):

```yaml
providers:
  - name: aws
    regions:
      - us-east-1
      - us-west-2

resources:
  include_all: true
  relationships: true

export:
  format: json
  pretty: true
```

2. Run the inspector:

```bash
./pmp-cloud-inspector inspect -c config.yaml -o output.json
```

## Usage

PMP Cloud Inspector provides two main commands:

### `inspect` - Inspect and Export Resources

Inspect cloud resources and export them to various formats.

```bash
pmp-cloud-inspector inspect [flags]
```

**Flags:**
- `-c, --config string`: Path to configuration file (default "config.yaml")
- `-o, --output string`: Output file (defaults to stdout)
- `-f, --format string`: Output format: json, yaml, dot (overrides config)
- `-p, --pretty`: Pretty print output (default true)
- `--include-raw`: Include raw cloud provider data
- `--concurrent int`: Number of concurrent goroutines for parallel resource collection (default 4)

**Filter Flags:**
- `--filter-tag strings`: Filter by tags (e.g., `Environment=prod`, `Name~test`, `Owner`)
- `--filter-regex strings`: Filter by regex (e.g., `name:/prod-.*/`, `id:/^i-/`)
- `--filter-date strings`: Filter by date range (e.g., `created:>2024-01-01`, `updated:2024-01..2024-12`)
- `--filter-state string`: Filter by resource states (comma-separated, e.g., `running,active`)
- `--filter-property strings`: Filter by property (e.g., `vm_size=Standard_D2s_v3`, `enabled=true`, `logins_count>100`)
- `--filter-cost string`: Filter by cost (e.g., `100..500`, `>100`, `<500`)
- `--filter-type strings`: Filter by resource types (e.g., `aws:ec2:instance`)
- `--filter-provider strings`: Filter by providers (e.g., `aws`, `azure`, `gcp`)

**Examples:**

Export all AWS resources to JSON:
```bash
pmp-cloud-inspector inspect -c config.yaml -o resources.json
```

Export specific resource types to YAML:
```bash
pmp-cloud-inspector inspect -c config.yaml -f yaml -o iam-resources.yaml
```

Generate a GraphViz visualization:
```bash
pmp-cloud-inspector inspect -c config.yaml -f dot -o resources.dot
dot -Tpng resources.dot -o resources.png
```

Use concurrent collection for faster resource gathering across multiple regions:
```bash
pmp-cloud-inspector inspect -c config.yaml --concurrent 8 -o resources.json
```

**Advanced Filtering Examples:**

Filter resources by tags:
```bash
# Resources with Environment tag set to production
pmp-cloud-inspector inspect -c config.yaml --filter-tag Environment=prod

# Resources with Environment tag containing "prod"
pmp-cloud-inspector inspect -c config.yaml --filter-tag "Environment~prod"

# Resources that have an Owner tag (any value)
pmp-cloud-inspector inspect -c config.yaml --filter-tag Owner
```

Filter resources by regex patterns:
```bash
# VMs with names starting with "prod-"
pmp-cloud-inspector inspect -c config.yaml --filter-regex "name:/^prod-.*/"

# AWS EC2 instances only
pmp-cloud-inspector inspect -c config.yaml --filter-regex "id:/^i-[0-9a-f]+$/"
```

Filter by resource state:
```bash
# Only running or active resources
pmp-cloud-inspector inspect -c config.yaml --filter-state running,active
```

Filter by properties:
```bash
# Azure VMs with specific size
pmp-cloud-inspector inspect -c config.yaml --filter-property "vm_size=Standard_D2s_v3"

# Resources with login count greater than 100
pmp-cloud-inspector inspect -c config.yaml --filter-property "logins_count>100"

# Enabled resources
pmp-cloud-inspector inspect -c config.yaml --filter-property "enabled=true"
```

Filter by date range:
```bash
# Resources created after 2024-01-01
pmp-cloud-inspector inspect -c config.yaml --filter-date "created:>2024-01-01"

# Resources updated in January 2024
pmp-cloud-inspector inspect -c config.yaml --filter-date "updated:2024-01-01..2024-01-31"
```

Combine multiple filters (all filters are ANDed):
```bash
# Production AWS EC2 instances created in 2024
pmp-cloud-inspector inspect -c config.yaml \
  --filter-tag Environment=production \
  --filter-type aws:ec2:instance \
  --filter-date "created:>2024-01-01" \
  -o production-ec2.json
```

### `ui` - Web Interface

Start a web server with a beautiful UI for viewing cloud resources.

```bash
pmp-cloud-inspector ui [flags]
```

**Flags:**
- `-p, --port int`: Port to listen on (default 8080)

**Examples:**

Start the UI on default port 8080:
```bash
pmp-cloud-inspector ui
```

Start the UI on custom port:
```bash
pmp-cloud-inspector ui -p 3000
```

Then open your browser at `http://localhost:8080` and upload your exported JSON or YAML files to view and explore your cloud resources interactively.

**UI Features:**
- Full-text search across all resource attributes
- Filter by provider, type, region
- Sort by name, type, or cost
- Group resources by provider, type, region, tags, or cost range
- Interactive D3.js graph visualization
- Resource details modal with drill-down
- **Cost visualization with breakdown charts**
- Min/Max cost filtering

### Cost Estimation

The tool includes built-in cost estimation for cloud resources. When enabled, it provides monthly cost estimates for each resource based on simplified pricing models.

**Enable cost estimation:**
```bash
pmp-cloud-inspector inspect -c config.yaml --estimate-costs -o resources.json
```

**Cost Features:**
- Estimated monthly costs for 15+ AWS resource types
- Estimated monthly costs for 6 Azure resource types
- Estimated monthly costs for 5 GCP resource types
- Instance/VM size-based cost multipliers
- Stopped/terminated resource detection (zero cost)
- Cost aggregations by provider, region, type, and tags
- Cost breakdowns by component (compute, storage, etc.)

**UI Cost Visualization:**
- Monthly cost summary card
- Cost badges on each resource card
- Cost filtering (min/max range)
- Sort by cost (high to low, low to high)
- Group by cost ranges
- Interactive pie charts showing:
  - Cost by Provider
  - Cost by Region (top 10)
  - Cost by Resource Type (top 10)
- Detailed cost breakdown in resource modal

**Example output with costs:**
```json
{
  "resources": [{
    "id": "i-1234567890",
    "name": "my-instance",
    "type": "aws:ec2:instance",
    "cost": {
      "monthly_estimate": 30.37,
      "currency": "USD",
      "breakdown": {
        "compute": 30.37
      }
    }
  }],
  "metadata": {
    "total_cost": {
      "total": 1234.56,
      "currency": "USD",
      "by_provider": {
        "aws": 800.50,
        "azure": 300.06
      },
      "by_region": {
        "us-east-1": 500.00
      }
    }
  }
}
```

**Note:** Cost estimates use simplified pricing models based on industry averages. For production use cases requiring accurate real-time pricing, integrate with:
- AWS Cost Explorer API
- Azure Cost Management API
- GCP Cloud Billing Catalog API

### `compare` - Compare Exports and Detect Drift

Compare two cloud resource exports to identify changes between different points in time.

```bash
pmp-cloud-inspector compare [flags]
```

**Flags:**
- `-b, --base string`: Base export file (older snapshot) [required]
- `-c, --compare string`: Compare export file (newer snapshot) [required]
- `-t, --type string`: Output type: summary, detailed, json (default "summary")

**Examples:**

Compare two exports with summary view:
```bash
pmp-cloud-inspector compare -b yesterday.json -c today.json
```

Get detailed changes:
```bash
pmp-cloud-inspector compare -b yesterday.json -c today.json -t detailed
```

Get JSON output for programmatic use:
```bash
pmp-cloud-inspector compare -b yesterday.json -c today.json -t json
```

The compare command shows:
- **Added resources**: Resources that exist in the new export but not in the old one
- **Removed resources**: Resources that existed in the old export but not in the new one
- **Modified resources**: Resources that exist in both but have changed properties
- **Unchanged resources**: Resources that are identical in both exports

## Configuration

The configuration file uses YAML format with three main sections:

### Providers

Specify which cloud providers to inspect:

```yaml
providers:
  - name: aws
    accounts: []  # Empty = all available accounts
    regions:
      - us-east-1
      - us-west-2
    rate_limit_ms: 0  # Optional: delay between API calls in milliseconds (0 = no rate limiting)
    options: {}
```

**Rate Limiting:**

To avoid hitting cloud provider API rate limits, you can configure a delay between API calls using the `rate_limit_ms` option:

```yaml
providers:
  - name: aws
    regions:
      - us-east-1
    rate_limit_ms: 100  # Wait 100ms between each API call
```

Common rate limit values:
- `0` (default): No rate limiting - maximum speed
- `50-100`: Light rate limiting for most use cases
- `200-500`: Moderate rate limiting for accounts with many resources
- `1000+`: Heavy rate limiting for conservative use or strict API limits

Note: Rate limiting applies to each individual API call within the provider, which helps prevent throttling errors from cloud providers while maintaining reasonable collection speeds.

### Resources

Control which resources to collect:

```yaml
resources:
  # Specific resource types
  types:
    - aws:iam:user
    - aws:ec2:vpc

  # Or collect all types
  include_all: true

  # Discover relationships
  relationships: true
```

Available AWS resource types:
- `aws:iam:user`
- `aws:iam:role`
- `aws:account`
- `aws:ec2:vpc`
- `aws:ec2:subnet`
- `aws:ec2:security-group`
- `aws:ec2:instance`
- `aws:ecr:repository`
- `aws:eks:cluster`
- `aws:elb:classic`
- `aws:elb:application`
- `aws:elb:network`
- `aws:lambda:function`
- `aws:apigateway:api`
- `aws:cloudfront:distribution`
- `aws:memorydb:cluster`
- `aws:elasticache:cluster`
- `aws:secretsmanager:secret`
- `aws:sns:topic`
- `aws:sqs:queue`
- `aws:dynamodb:table`

Available GitHub resource types:
- `github:organization`
- `github:repository`
- `github:team`
- `github:user`

### Export

Configure output format and options:

```yaml
export:
  format: json        # json, yaml, or dot
  output_file: ""     # Path to output file (optional)
  pretty: true        # Pretty print output
  include_raw: false  # Include raw cloud provider data
```

## Architecture

### Provider Interface

The tool uses a provider trait pattern that makes it easy to add new cloud providers:

```go
type Provider interface {
    Name() string
    Initialize(ctx context.Context, config config.ProviderConfig) error
    GetSupportedResourceTypes() []resource.ResourceType
    CollectResources(ctx context.Context, types []resource.ResourceType) (*resource.Collection, error)
    DiscoverRelationships(ctx context.Context, collection *resource.Collection) error
    GetAccounts(ctx context.Context) ([]string, error)
    GetRegions(ctx context.Context) ([]string, error)
}
```

### Resource Model

Resources are represented with a common structure:

```go
type Resource struct {
    ID           string
    Type         ResourceType
    Name         string
    Provider     string
    Account      string
    Region       string
    ARN          string
    Tags         map[string]string
    Properties   map[string]interface{}
    RawData      interface{}
    Relationships []Relationship
    CreatedAt    *time.Time
    UpdatedAt    *time.Time
}
```

### Relationships

Resources can have relationships with each other:

- `contains`: e.g., VPC contains Subnets
- `belongs_to`: e.g., Subnet belongs to VPC
- `attached_to`: e.g., SecurityGroup attached to Instance
- `assumes`: e.g., Service assumes Role
- `has_access`: e.g., User has access to Resource
- `references`: Generic reference
- `depends_on`: Dependency relationship

## Adding New Providers

To add a new provider:

1. Create a new package in `pkg/provider/<provider-name>/`
2. Implement the `Provider` interface
3. Register your provider in the `init()` function
4. Add resource type constants
5. Implement resource collectors

See `pkg/provider/aws/` for a complete example.

## Output Formats

### JSON
Standard JSON format with all resource data and metadata.

### YAML
Human-readable YAML format.

### DOT (GraphViz)
Graph visualization format showing resources and their relationships. Can be converted to images using GraphViz:

```bash
dot -Tpng resources.dot -o resources.png
```

## Provider Authentication

All provider credentials are configured using environment variables for security.

### AWS Authentication

The AWS provider uses the standard AWS SDK credential chain:

1. **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` (optional)
2. **Shared Credentials File**: `~/.aws/credentials`
3. **IAM Role**: When running on EC2 or other AWS services

**Environment Variables:**
```bash
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_REGION="us-east-1"  # Optional: default region
```

**Config file:**
```yaml
providers:
  - name: aws
    regions:
      - us-east-1
      - us-west-2
```

See the [Provider Permissions Reference](#provider-permissions-reference) section for required IAM permissions.

### GitHub Authentication

The GitHub provider requires a personal access token.

**Setup:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select scopes:
   - `read:org` - Read organization data
   - `repo` - Access repositories (for private repos)
   - `admin:org` - Read organization teams and members

**Environment Variable:**
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**Config file** (only specify accounts to inspect):
```yaml
providers:
  - name: github
    accounts:
      - my-organization
```

### GitLab Authentication

The GitLab provider requires a personal access token.

**Setup:**
1. Go to GitLab Settings → Access Tokens
2. Create a new token with scopes:
   - `read_api` - Read API access
   - `read_repository` - Read repository data

**Environment Variables:**
```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
export GITLAB_BASE_URL="https://gitlab.com"  # Optional: for self-hosted GitLab
```

**Config file:**
```yaml
providers:
  - name: gitlab
    accounts:
      - my-group  # Optional: specific groups to inspect
```

### JFrog Artifactory Authentication

The JFrog provider requires either an API key or username/password.

**Environment Variables (API Key):**
```bash
export JFROG_BASE_URL="https://mycompany.jfrog.io"
export JFROG_API_KEY="AKCxxxxxxxxxxxxxxxxxxxx"
```

**Or with Username/Password:**
```bash
export JFROG_BASE_URL="https://mycompany.jfrog.io"
export JFROG_USERNAME="admin"
export JFROG_PASSWORD="password"
```

**Config file:**
```yaml
providers:
  - name: jfrog
```

### GCP Authentication

The GCP provider uses Application Default Credentials or a service account key file.

**Setup:**
1. Create a service account in GCP Console
2. Grant necessary IAM roles (Viewer or custom roles)
3. Download the JSON key file

**Environment Variables:**
```bash
export GCP_PROJECT_ID="my-gcp-project"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"  # Optional: uses ADC if not provided
```

**Config file:**
```yaml
providers:
  - name: gcp
    regions:
      - us-central1
      - us-east1
```

Required GCP IAM permissions:
- `compute.instances.list`
- `compute.networks.list`
- `compute.subnetworks.list`
- `storage.buckets.list`
- `cloudfunctions.functions.list`
- `run.services.list`

### Okta Authentication

The Okta provider requires an API token.

**Setup:**
1. Go to Okta Admin Console → Security → API → Tokens
2. Click "Create Token"
3. Give it a descriptive name (e.g., "PMP Cloud Inspector")
4. Copy the token value (you won't be able to see it again)

**Environment Variables:**
```bash
export OKTA_ORG_URL="https://your-domain.okta.com"
export OKTA_API_TOKEN="00xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**Config file:**
```yaml
providers:
  - name: okta
```

Required Okta API scopes (automatically included with API tokens):
- `okta.users.read` - Read user information
- `okta.groups.read` - Read group information
- `okta.apps.read` - Read application information
- `okta.authorizationServers.read` - Read authorization server information

### Auth0 Authentication

The Auth0 provider supports two authentication methods: Client Credentials or Management API Token.

**Method 1: Client Credentials (Recommended)**

**Setup:**
1. Go to Auth0 Dashboard → Applications → Applications
2. Create a Machine to Machine Application
3. Authorize it for the Auth0 Management API
4. Grant the following permissions (scopes):
   - `read:users` - Read user information
   - `read:roles` - Read role information
   - `read:clients` - Read client/application information
   - `read:resource_servers` - Read resource server/API information
   - `read:connections` - Read connection information

**Environment Variables:**
```bash
export AUTH0_DOMAIN="your-tenant.us.auth0.com"
export AUTH0_CLIENT_ID="xxxxxxxxxxxxxxxxxxxx"
export AUTH0_CLIENT_SECRET="yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
```

**Method 2: Management API Token**

**Setup:**
1. Go to Auth0 Dashboard → Applications → APIs → Auth0 Management API
2. Go to API Explorer tab
3. Create a token with the required scopes listed above

**Environment Variables:**
```bash
export AUTH0_DOMAIN="your-tenant.us.auth0.com"
export AUTH0_MANAGEMENT_API_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6..."
```

**Config file:**
```yaml
providers:
  - name: auth0
```

### Azure Authentication

The Azure provider uses the DefaultAzureCredential authentication flow, which supports multiple authentication methods in order:

1. **Environment Variables** (Recommended for automation)
2. **Managed Identity** (When running on Azure resources)
3. **Azure CLI** (When logged in via `az login`)

**Setup with Environment Variables:**
1. Create a Service Principal:
   ```bash
   az ad sp create-for-rbac --name "pmp-cloud-inspector" --role Reader --scopes /subscriptions/{subscription-id}
   ```
2. The command will output the credentials you need

**Environment Variables:**
```bash
export AZURE_SUBSCRIPTION_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export AZURE_TENANT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"         # Optional
export AZURE_CLIENT_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"         # Optional
export AZURE_CLIENT_SECRET="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"       # Optional
```

**Or use Azure CLI authentication:**
```bash
az login
export AZURE_SUBSCRIPTION_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

**Config file:**
```yaml
providers:
  - name: azure
    regions:
      - eastus
      - westus2
```

Required Azure RBAC permissions (built-in Reader role or custom role with):
- `Microsoft.Resources/subscriptions/resourceGroups/read`
- `Microsoft.Compute/virtualMachines/read`
- `Microsoft.Network/virtualNetworks/read`
- `Microsoft.Network/virtualNetworks/subnets/read`
- `Microsoft.Storage/storageAccounts/read`
- `Microsoft.Web/sites/read`
- `Microsoft.Sql/servers/databases/read`
- `Microsoft.KeyVault/vaults/read`

## Provider Permissions Reference

This section provides a comprehensive reference of all permissions required by each provider.

### AWS Required Permissions

The AWS provider requires the following IAM permissions. You can use the built-in `ReadOnlyAccess` policy or create a custom policy with these specific permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "iam:ListUsers",
        "iam:GetUser",
        "iam:ListRoles",
        "iam:GetRole",
        "iam:ListAttachedUserPolicies",
        "iam:ListAttachedRolePolicies",
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeInstances",
        "ec2:DescribeRegions",
        "ecr:DescribeRepositories",
        "ecr:ListTagsForResource",
        "eks:ListClusters",
        "eks:DescribeCluster",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTags",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeListeners",
        "lambda:ListFunctions",
        "lambda:GetFunction",
        "lambda:ListTags",
        "apigateway:GET",
        "cloudfront:ListDistributions",
        "cloudfront:GetDistribution",
        "memorydb:DescribeClusters",
        "elasticache:DescribeCacheClusters",
        "elasticache:ListTagsForResource",
        "secretsmanager:ListSecrets",
        "secretsmanager:DescribeSecret",
        "sns:ListTopics",
        "sns:GetTopicAttributes",
        "sns:ListTagsForResource",
        "sqs:ListQueues",
        "sqs:GetQueueAttributes",
        "sqs:ListQueueTags",
        "dynamodb:ListTables",
        "dynamodb:DescribeTable",
        "dynamodb:ListTagsOfResource",
        "sts:GetCallerIdentity",
        "organizations:DescribeAccount"
      ],
      "Resource": "*"
    }
  ]
}
```

**Minimum IAM Policy:**
Create a custom IAM policy with the above permissions and attach it to your IAM user or role.

### GitHub Required Permissions

The GitHub provider requires a Personal Access Token with the following scopes:

**Token Scopes:**
- `read:org` - Read organization membership, teams, and settings
- `repo` - Full control of private repositories (only needed if inspecting private repos)
- `public_repo` - Access public repositories (alternative to `repo` for public-only)
- `admin:org` → `read:org` - Read organization data, teams, and members

**Minimum Scopes for Public Repos Only:**
- `read:org`
- `public_repo`

**For Private Repositories:**
- `read:org`
- `repo`

### GitLab Required Permissions

The GitLab provider requires a Personal Access Token with the following scopes:

**Token Scopes:**
- `read_api` - Read-only API access (grants access to most read operations)
- `read_repository` - Read repository content (needed for repository inspection)

**Additional Permissions for Self-Hosted:**
- Ensure the token user has at least Reporter role on groups/projects to inspect

### GCP Required Permissions

The GCP provider requires a Service Account with the following IAM permissions. You can use the built-in `Viewer` role or create a custom role:

**Required Permissions:**
- `compute.instances.list` - List compute instances
- `compute.instances.get` - Get compute instance details
- `compute.networks.list` - List VPC networks
- `compute.networks.get` - Get VPC network details
- `compute.subnetworks.list` - List subnetworks
- `compute.subnetworks.get` - Get subnetwork details
- `storage.buckets.list` - List storage buckets
- `storage.buckets.get` - Get storage bucket details
- `cloudfunctions.functions.list` - List cloud functions
- `cloudfunctions.functions.get` - Get cloud function details
- `run.services.list` - List Cloud Run services
- `run.services.get` - Get Cloud Run service details
- `resourcemanager.projects.get` - Get project metadata

**Recommended IAM Role:**
- Use the built-in `roles/viewer` role, or
- Create a custom role with only the permissions listed above

### JFrog Artifactory Required Permissions

The JFrog provider requires either an API Key or username/password with the following permissions:

**Required Permissions:**
- Read access to all repositories
- Read access to user management
- Read access to group management
- Read access to permission targets

**Recommended Setup:**
- Use an API Key associated with an admin user, or
- Create a dedicated user with read-only admin privileges

### Okta Required Permissions

The Okta provider requires an API Token. API tokens automatically inherit permissions from the admin user who creates them.

**Required API Scopes:**
- `okta.users.read` - Read user information and profiles
- `okta.groups.read` - Read group information and membership
- `okta.apps.read` - Read application configurations
- `okta.authorizationServers.read` - Read authorization server configurations

**Recommended Setup:**
1. Create a dedicated "service account" admin user
2. Assign the "Read-only Administrator" role to this user
3. Generate an API token from this user's account

### Auth0 Required Permissions

The Auth0 provider requires either Client Credentials (Machine-to-Machine app) or a Management API Token with the following scopes:

**Required Scopes:**
- `read:users` - Read user profiles and metadata
- `read:roles` - Read role definitions and assignments
- `read:clients` - Read application/client configurations
- `read:resource_servers` - Read API (Resource Server) configurations
- `read:connections` - Read identity provider connection configurations

**Recommended Setup:**
- Use a Machine-to-Machine application (Client Credentials flow)
- Authorize it for the Auth0 Management API
- Grant only the read scopes listed above (principle of least privilege)

### Azure Required Permissions

The Azure provider requires a Service Principal or Managed Identity with the following permissions:

**Required RBAC Permissions:**
Use the built-in `Reader` role at the subscription scope, or create a custom role with these permissions:

- `Microsoft.Resources/subscriptions/read` - Read subscription information
- `Microsoft.Resources/subscriptions/resourceGroups/read` - List and read resource groups
- `Microsoft.Compute/virtualMachines/read` - Read virtual machine configurations
- `Microsoft.Compute/virtualMachines/instanceView/read` - Read VM runtime state
- `Microsoft.Network/virtualNetworks/read` - Read virtual networks
- `Microsoft.Network/virtualNetworks/subnets/read` - Read subnets
- `Microsoft.Storage/storageAccounts/read` - Read storage account configurations
- `Microsoft.Web/sites/read` - Read App Service configurations
- `Microsoft.Sql/servers/read` - Read SQL server configurations
- `Microsoft.Sql/servers/databases/read` - Read SQL database configurations
- `Microsoft.KeyVault/vaults/read` - Read Key Vault configurations

**Recommended Setup:**
1. Create a Service Principal: `az ad sp create-for-rbac --name "pmp-cloud-inspector" --role Reader --scopes /subscriptions/{subscription-id}`
2. Use the `Reader` role at the subscription level for full read access
3. Or assign specific resource group scopes if you want to limit access

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See LICENSE file for details.

## Roadmap

### Completed
- [x] GCP provider support
- [x] GitLab provider support
- [x] JFrog Artifactory provider support
- [x] Okta provider support
- [x] Auth0 provider support
- [x] Azure provider support
- [x] Additional AWS resource types (SNS, SQS, DynamoDB)
- [x] Resource drift detection and comparison
- [x] Web UI for viewing and comparing exports
- [x] Multi-platform binary releases with GoReleaser
- [x] Advanced filtering and querying (tags, regex, dates, properties, costs)
- [x] Concurrent resource collection for improved performance
- [x] Full-text search across all resource attributes
- [x] D3.js graph visualization of resource relationships
- [x] Resource grouping in UI (by provider, type, region, tags, cost)
- [x] **Cost estimation and tracking** with UI visualization

### Planned / Future Enhancements
- [ ] Additional AWS resource types (RDS, S3, CloudWatch, Step Functions, ECS, Fargate, etc.)
- [ ] Real-time cost API integration (AWS Cost Explorer, Azure Cost Management, GCP Billing)
- [ ] Historical cost tracking and trend analysis
- [ ] Security compliance checks (CIS benchmarks, security best practices)
- [ ] Resource tagging recommendations
- [ ] Export to Infrastructure-as-Code (Terraform, CloudFormation, Pulumi)
- [ ] Historical tracking and trend analysis
- [ ] Automated scheduling and continuous monitoring
- [ ] Slack/Teams/Email notifications for drift detection
- [ ] RBAC and multi-user support in UI
- [ ] Resource optimization recommendations

## CI/CD

The project includes comprehensive GitHub Actions workflows:

- **PR Checks** (`.github/workflows/pr.yml`): Runs on pull requests to main
  - Linting with golangci-lint
  - Tests with race detector (`-race --count=1`)
  - Build verification

- **Main Branch** (`.github/workflows/main.yml`): Runs on merges to main
  - Linting with golangci-lint
  - Tests with race detector (`-race --count=1`)
  - Build verification

- **Release** (`.github/workflows/release.yml`): Triggers on semantic version tags (v*.*.*)
  - Linting with golangci-lint
  - Tests with race detector (`-race --count=1`)
  - Multi-platform binary builds (Linux, macOS, Windows)
  - Automatic GitHub release creation with binaries and checksums
  - Changelog generation

To create a release, simply push a tag with semantic versioning:
```bash
git tag v1.0.0
git push origin v1.0.0
```
