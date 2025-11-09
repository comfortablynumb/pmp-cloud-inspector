package resource

// Graph represents a graph of resources and their relationships
type Graph struct {
	Collection *Collection
	edges      map[string][]string // adjacency list: resourceID -> []relatedResourceIDs
}

// NewGraph creates a new resource graph
func NewGraph(collection *Collection) *Graph {
	g := &Graph{
		Collection: collection,
		edges:      make(map[string][]string),
	}
	g.buildGraph()
	return g
}

// buildGraph constructs the graph from resource relationships
func (g *Graph) buildGraph() {
	for _, resource := range g.Collection.Resources {
		g.edges[resource.ID] = make([]string, 0)

		for _, rel := range resource.Relationships {
			g.edges[resource.ID] = append(g.edges[resource.ID], rel.TargetID)
		}
	}
}

// GetRelated returns all resources related to the given resource ID
func (g *Graph) GetRelated(id string) []*Resource {
	relatedIDs := g.edges[id]
	related := make([]*Resource, 0, len(relatedIDs))

	for _, relatedID := range relatedIDs {
		if resource := g.Collection.Get(relatedID); resource != nil {
			related = append(related, resource)
		}
	}

	return related
}

// GetRelationships returns all relationships for a given resource
func (g *Graph) GetRelationships(id string) []Relationship {
	resource := g.Collection.Get(id)
	if resource == nil {
		return nil
	}
	return resource.Relationships
}

// AddRelationship adds a relationship between two resources
func (g *Graph) AddRelationship(fromID string, rel Relationship) {
	resource := g.Collection.Get(fromID)
	if resource == nil {
		return
	}

	// Add to resource
	resource.Relationships = append(resource.Relationships, rel)

	// Update edges
	g.edges[fromID] = append(g.edges[fromID], rel.TargetID)
}

// GetSubgraph returns a subgraph containing only resources of specified types
func (g *Graph) GetSubgraph(types ...ResourceType) *Graph {
	typeSet := make(map[ResourceType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	subCollection := NewCollection()
	for _, resource := range g.Collection.Resources {
		if typeSet[resource.Type] {
			// Create a copy of the resource with filtered relationships
			filtered := *resource
			filtered.Relationships = make([]Relationship, 0)

			for _, rel := range resource.Relationships {
				if typeSet[rel.TargetType] {
					filtered.Relationships = append(filtered.Relationships, rel)
				}
			}

			subCollection.Add(&filtered)
		}
	}

	return NewGraph(subCollection)
}
