package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	elasticacheTypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	memorydbTypes "github.com/aws/aws-sdk-go-v2/service/memorydb/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectMemoryDBClusters collects all MemoryDB clusters in a region
func (p *Provider) collectMemoryDBClusters(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting MemoryDB clusters in %s...\n", region)
	client := memorydb.NewFromConfig(cfg)

	paginator := memorydb.NewDescribeClustersPaginator(client, &memorydb.DescribeClustersInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe MemoryDB clusters: %w", err)
		}

		for _, cluster := range output.Clusters {
			res := p.convertMemoryDBClusterToResource(&cluster, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found MemoryDB cluster: %s (%s)\n", safeString(cluster.Name), safeString(cluster.Status))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d MemoryDB clusters in %s\n", count, region)
	return nil
}

// collectElastiCacheClusters collects all ElastiCache clusters in a region
func (p *Provider) collectElastiCacheClusters(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting ElastiCache clusters in %s...\n", region)
	client := elasticache.NewFromConfig(cfg)

	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe ElastiCache clusters: %w", err)
		}

		for _, cluster := range output.CacheClusters {
			res := p.convertElastiCacheClusterToResource(&cluster, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found ElastiCache cluster: %s (%s)\n", safeString(cluster.CacheClusterId), safeString(cluster.CacheClusterStatus))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d ElastiCache clusters in %s\n", count, region)
	return nil
}

// convertMemoryDBClusterToResource converts a MemoryDB cluster to a Resource
func (p *Provider) convertMemoryDBClusterToResource(cluster *memorydbTypes.Cluster, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"status":      safeString(cluster.Status),
		"node_type":   safeString(cluster.NodeType),
		"engine":      safeString(cluster.Engine),
		"num_shards":  safeInt32(cluster.NumberOfShards),
		"tls_enabled": safeBool(cluster.TLSEnabled),
	}

	if cluster.Description != nil {
		properties["description"] = *cluster.Description
	}
	if cluster.EngineVersion != nil {
		properties["engine_version"] = *cluster.EngineVersion
	}
	if cluster.SubnetGroupName != nil {
		properties["subnet_group_name"] = *cluster.SubnetGroupName
	}
	if cluster.ParameterGroupName != nil {
		properties["parameter_group_name"] = *cluster.ParameterGroupName
	}
	if cluster.ClusterEndpoint != nil {
		properties["endpoint"] = map[string]interface{}{
			"address": safeString(cluster.ClusterEndpoint.Address),
			"port":    cluster.ClusterEndpoint.Port,
		}
	}

	res := &resource.Resource{
		ID:         safeString(cluster.Name),
		Type:       resource.TypeAWSMemoryDB,
		Name:       safeString(cluster.Name),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(cluster.ARN),
		Properties: properties,
		RawData:    cluster,
	}

	return res
}

// convertElastiCacheClusterToResource converts an ElastiCache cluster to a Resource
func (p *Provider) convertElastiCacheClusterToResource(cluster *elasticacheTypes.CacheCluster, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"status":          safeString(cluster.CacheClusterStatus),
		"node_type":       safeString(cluster.CacheNodeType),
		"engine":          safeString(cluster.Engine),
		"num_cache_nodes": safeInt32(cluster.NumCacheNodes),
		"preferred_az":    safeString(cluster.PreferredAvailabilityZone),
	}

	if cluster.EngineVersion != nil {
		properties["engine_version"] = *cluster.EngineVersion
	}
	if cluster.CacheSubnetGroupName != nil {
		properties["subnet_group_name"] = *cluster.CacheSubnetGroupName
	}
	if cluster.CacheParameterGroup != nil && cluster.CacheParameterGroup.CacheParameterGroupName != nil {
		properties["parameter_group_name"] = *cluster.CacheParameterGroup.CacheParameterGroupName
	}
	if cluster.ConfigurationEndpoint != nil {
		properties["endpoint"] = map[string]interface{}{
			"address": safeString(cluster.ConfigurationEndpoint.Address),
			"port":    safeInt32(cluster.ConfigurationEndpoint.Port),
		}
	}

	arn := safeString(cluster.ARN)
	if arn == "" {
		// Construct ARN if not provided
		arn = fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, account, safeString(cluster.CacheClusterId))
	}

	res := &resource.Resource{
		ID:         safeString(cluster.CacheClusterId),
		Type:       resource.TypeAWSElastiCache,
		Name:       safeString(cluster.CacheClusterId),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Properties: properties,
		RawData:    cluster,
	}

	if cluster.CacheClusterCreateTime != nil {
		res.CreatedAt = cluster.CacheClusterCreateTime
	}

	return res
}

// safeInt32 safely dereferences an int32 pointer
func safeInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}
