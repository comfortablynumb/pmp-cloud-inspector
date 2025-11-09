# PMP Cloud Inspector

**PMP Cloud Inspector** is a powerful CLI tool that allows you to inspect your cloud accounts and export found resources to several formats. Part of the Poor Man's Platform ecosystem.

## Features

- **Multi-Provider Support**: Extensible provider architecture supporting AWS (with more providers coming soon)
- **Comprehensive Resource Discovery**: Automatically discovers and catalogs cloud resources
- **Relationship Mapping**: Discovers and tracks relationships between resources
- **Multiple Export Formats**: Export to JSON, YAML, or GraphViz DOT format
- **Flexible Configuration**: YAML-based configuration for fine-grained control
- **Resource Filtering**: Select specific resource types or collect all available resources

## Supported Providers

### AWS
- IAM Users
- IAM Roles
- Accounts
- VPCs
- Subnets
- Security Groups
- ECR Repositories

### GitHub
- Organizations
- Repositories
- Teams
- Users

More providers (GCP, Okta, JFrog, etc.) coming soon!

## Installation

### From Source

```bash
git clone https://github.com/comfortablynumb/pmp-cloud-inspector.git
cd pmp-cloud-inspector
go build -o pmp-cloud-inspector ./cmd/inspector
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
./pmp-cloud-inspector -c config.yaml -o output.json
```

## Usage

```bash
pmp-cloud-inspector [flags]
```

### Flags

- `-c, --config string`: Path to configuration file (default "config.yaml")
- `-o, --output string`: Output file (defaults to stdout)
- `-f, --format string`: Output format: json, yaml, dot (overrides config)
- `-p, --pretty`: Pretty print output (default true)
- `--include-raw`: Include raw cloud provider data

### Examples

**Export all AWS resources to JSON:**
```bash
pmp-cloud-inspector -c config.yaml -o resources.json
```

**Export specific resource types to YAML:**
```yaml
# config.yaml
resources:
  types:
    - aws:iam:user
    - aws:iam:role
  include_all: false
```

```bash
pmp-cloud-inspector -c config.yaml -f yaml -o iam-resources.yaml
```

**Generate a GraphViz visualization:**
```bash
pmp-cloud-inspector -c config.yaml -f dot -o resources.dot
dot -Tpng resources.dot -o resources.png
```

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
- `aws:ecr:repository`

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
- `ec2:DescribeRegions`
- `ecr:DescribeRepositories`
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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See LICENSE file for details.

## Roadmap

- [ ] GCP provider support
- [ ] Azure provider support
- [ ] Okta provider support
- [ ] JFrog provider support
- [ ] More AWS resource types (EC2, RDS, S3, etc.)
- [ ] Advanced filtering and querying
- [ ] Resource change detection
- [ ] Cost estimation
- [ ] Security compliance checks
