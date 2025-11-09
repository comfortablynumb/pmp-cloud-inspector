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

More providers (Azure, Okta, etc.) coming soon!

## Installation

### From Source

```bash
git clone https://github.com/comfortablynumb/pmp-cloud-inspector.git
cd pmp-cloud-inspector

# Basic build (AWS and GitHub providers only)
go build -o pmp-cloud-inspector ./cmd/inspector

# Build with additional providers (requires downloading dependencies)
go mod tidy
go build -tags "gitlab jfrog gcp" -o pmp-cloud-inspector ./cmd/inspector

# Or build with specific providers only
go build -tags "gitlab" -o pmp-cloud-inspector ./cmd/inspector  # GitLab only
go build -tags "gcp" -o pmp-cloud-inspector ./cmd/inspector     # GCP only
go build -tags "gitlab gcp" -o pmp-cloud-inspector ./cmd/inspector  # Multiple
```

### Prerequisites

- Go 1.21 or higher
- AWS credentials configured (for AWS provider)

### Pre-built Binaries

Download the latest pre-built binaries from the [Releases](https://github.com/comfortablynumb/pmp-cloud-inspector/releases) page.

Available for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

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
    options: {}
```

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

## AWS Authentication

The AWS provider uses the standard AWS SDK credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (when running on EC2)

Required IAM permissions:
- `iam:ListUsers`
- `iam:ListRoles`
- `ec2:DescribeVpcs`
- `ec2:DescribeSubnets`
- `ec2:DescribeSecurityGroups`
- `ec2:DescribeInstances`
- `ec2:DescribeRegions`
- `ecr:DescribeRepositories`
- `eks:ListClusters`
- `eks:DescribeCluster`
- `elasticloadbalancing:DescribeLoadBalancers`
- `elasticloadbalancing:DescribeTags`
- `lambda:ListFunctions`
- `apigateway:GET`
- `cloudfront:ListDistributions`
- `memorydb:DescribeClusters`
- `elasticache:DescribeCacheClusters`
- `secretsmanager:ListSecrets`
- `sts:GetCallerIdentity`

## GitHub Authentication

The GitHub provider requires a personal access token for authentication.

To create a personal access token:
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a descriptive name
4. Select scopes (minimum required):
   - `read:org` - Read organization data
   - `repo` - Access repositories (for private repos)
   - `admin:org` - Read organization teams and members

Configure the token in your config file:

```yaml
providers:
  - name: github
    accounts:
      - my-organization
    options:
      token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

## GitLab Authentication

The GitLab provider requires a personal access token for authentication.

To create a personal access token:
1. Go to GitLab Settings → Access Tokens
2. Create a new token with the following scopes:
   - `read_api` - Read API access
   - `read_repository` - Read repository data

Configure the token in your config file:

```yaml
providers:
  - name: gitlab
    accounts:
      - my-group  # Optional: specific groups to inspect
    options:
      token: "glpat-xxxxxxxxxxxxxxxxxxxx"
      base_url: "https://gitlab.com"  # Optional: for self-hosted GitLab
```

## JFrog Artifactory Authentication

The JFrog provider requires either an API key or username/password for authentication.

Configure in your config file:

```yaml
providers:
  - name: jfrog
    options:
      base_url: "https://mycompany.jfrog.io"
      api_key: "AKCxxxxxxxxxxxxxxxxxxxx"
      # OR use username/password:
      # username: "admin"
      # password: "password"
```

## GCP Authentication

The GCP provider uses Application Default Credentials or a service account key file.

To set up authentication:
1. Create a service account in GCP Console
2. Grant necessary IAM roles (Viewer or custom roles)
3. Download the JSON key file

Configure in your config file:

```yaml
providers:
  - name: gcp
    regions:
      - us-central1
      - us-east1
    options:
      project_id: "my-gcp-project"
      credentials_file: "/path/to/service-account-key.json"  # Optional: uses ADC if not provided
```

Required GCP IAM permissions:
- `compute.instances.list`
- `compute.networks.list`
- `compute.subnetworks.list`
- `storage.buckets.list`
- `cloudfunctions.functions.list`
- `run.services.list`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See LICENSE file for details.

## Roadmap

- [ ] GCP provider support
- [ ] Azure provider support
- [ ] Okta provider support
- [ ] JFrog provider support
- [ ] Additional AWS resource types (RDS, S3, DynamoDB, SQS, SNS, etc.)
- [ ] Advanced filtering and querying
- [ ] Resource change detection
- [ ] Cost estimation
- [ ] Security compliance checks
