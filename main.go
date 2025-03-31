package main

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Update the Node struct to include both Level and Security fields
type Node struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Desc     string `json:"desc"`
	Security int    `json:"security"`
	Level    int    `json:"level,omitempty"` // Used internally for layout, but will be included in JSON
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Modify reorganizeGraphLayout to preserve security levels
func reorganizeGraphLayout(graph *GraphData) {
	if len(graph.Nodes) == 0 {
		return
	}

	// Create a map for node lookup
	nodeMap := make(map[string]*Node)
	for i := range graph.Nodes {
		nodeMap[graph.Nodes[i].ID] = &graph.Nodes[i]
	}

	// Store security values before BFS traversal
	securityValues := make(map[string]int)
	for _, node := range graph.Nodes {
		securityValues[node.ID] = node.Security
	}

	// Calculate incoming edges for each node
	incomingEdges := make(map[string]int)
	for _, edge := range graph.Edges {
		incomingEdges[edge.To]++
	}

	// Identify root nodes (nodes with no incoming edges)
	var rootNodes []string
	for _, node := range graph.Nodes {
		if incomingEdges[node.ID] == 0 {
			rootNodes = append(rootNodes, node.ID)
		}
	}

	// If no root nodes found, find nodes with minimum incoming edges
	if len(rootNodes) == 0 {
		minIncoming := len(graph.Edges) + 1
		for _, count := range incomingEdges {
			if count < minIncoming {
				minIncoming = count
			}
		}
		for id, count := range incomingEdges {
			if count == minIncoming {
				rootNodes = append(rootNodes, id)
			}
		}
	}

	// Build adjacency list for outgoing edges
	outgoingEdges := make(map[string][]string)
	for _, edge := range graph.Edges {
		outgoingEdges[edge.From] = append(outgoingEdges[edge.From], edge.To)
	}

	// Perform breadth-first traversal to assign levels to nodes
	levels := make(map[string]int)
	visited := make(map[string]bool)

	// Initialize queue with root nodes at level 0
	type QueueItem struct {
		ID    string
		Level int
	}
	queue := make([]QueueItem, 0)

	for _, rootID := range rootNodes {
		levels[rootID] = 0
		visited[rootID] = true
		queue = append(queue, QueueItem{ID: rootID, Level: 0})
	}

	// BFS to assign levels
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		for _, childID := range outgoingEdges[item.ID] {
			if !visited[childID] {
				visited[childID] = true
				levels[childID] = item.Level + 1
				queue = append(queue, QueueItem{ID: childID, Level: item.Level + 1})
			}
		}
	}

	// Calculate max level and nodes per level
	maxLevel := 0
	nodesPerLevel := make(map[int]int)
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
		nodesPerLevel[level]++
	}

	// Position nodes based on levels
	const (
		containerWidth  = 1000
		containerHeight = 550
		topMargin       = 50
		levelHeight     = 100
	)

	// Position nodes at each level
	for id, level := range levels {
		node := nodeMap[id]
		if node == nil {
			continue
		}

		// Calculate horizontal position based on the position in the level
		nodesInLevel := nodesPerLevel[level]
		horizontalSpacing := containerWidth / (nodesInLevel + 1)

		// Find position of this node within its level
		position := 1
		for otherId, otherLevel := range levels {
			if otherLevel == level && otherId < id {
				position++
			}
		}

		// Set the node coordinates and layout level
		node.X = position * horizontalSpacing
		node.Y = topMargin + level*levelHeight
		node.Level = level // This is for layout purposes
	}

	// For unvisited nodes (not connected to the main graph)
	extraY := topMargin + (maxLevel+1)*levelHeight
	extraCount := 0
	for _, node := range graph.Nodes {
		if !visited[node.ID] {
			extraCount++
			node := nodeMap[node.ID]
			node.X = extraCount * (containerWidth / (len(graph.Nodes) - len(visited) + 1))
			node.Y = extraY
			node.Level = maxLevel + 1 // Assign a level below the connected graph
		}
	}

	// Restore security values to ensure they weren't overwritten
	for i := range graph.Nodes {
		graph.Nodes[i].Security = securityValues[graph.Nodes[i].ID]
	}
}

func validateGraph(graph *GraphData) {
	// Create a set of valid node IDs
	validNodes := make(map[string]bool)
	for _, node := range graph.Nodes {
		validNodes[node.ID] = true
	}

	// Filter edges to keep only those with valid node references
	validEdges := []Edge{}
	for _, edge := range graph.Edges {
		if validNodes[edge.From] && validNodes[edge.To] {
			validEdges = append(validEdges, edge)
		}
	}

	// Update the graph with only valid edges
	graph.Edges = validEdges
}

// Add this function to remove isolated nodes (nodes with no connections)
func removeIsolatedNodes(graph *GraphData) {
	// Create a set of connected node IDs
	connectedNodes := make(map[string]bool)

	// Add all nodes that appear in any edge
	for _, edge := range graph.Edges {
		connectedNodes[edge.From] = true
		connectedNodes[edge.To] = true
	}

	// Filter the nodes to keep only those with connections
	connectedNodesList := []Node{}
	for _, node := range graph.Nodes {
		if connectedNodes[node.ID] {
			connectedNodesList = append(connectedNodesList, node)
		}
	}

	// Update the graph with only connected nodes
	graph.Nodes = connectedNodesList
}

// Update your handleGraphFile function to include better error reporting
func handleGraphFile(file *multipart.FileHeader) (GraphData, error) {
	opened, err := file.Open()
	if err != nil {
		return GraphData{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer opened.Close()

	var graph GraphData
	if err := json.NewDecoder(opened).Decode(&graph); err != nil {
		return GraphData{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Log node and edge counts for debugging
	fmt.Printf("Parsed graph with %d nodes and %d edges\n", len(graph.Nodes), len(graph.Edges))

	// Validate the structure of the graph
	if len(graph.Nodes) == 0 {
		return GraphData{}, fmt.Errorf("graph data is incomplete: no nodes found")
	}

	return graph, nil
}

// Add to your main.go file
func setupHtmxEndpoints(r *gin.Engine) {
	// Get node details endpoint
	r.GET("/node/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Fetch node details from your data store
		// For demonstration, we'll return dummy data
		c.HTML(http.StatusOK, "node_details.html", gin.H{
			"id":       id,
			"label":    "Node " + id,
			"desc":     "Details for node " + id,
			"security": 1,
		})
	})

	// Update upload endpoint to handle HTMX requests
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("jsonFile")
		if err != nil {
			if c.GetHeader("HX-Request") == "true" {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "File not received"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "File not received"})
			}
			return
		}

		// Process file as before
		graph, err := handleGraphFile(file)
		if err != nil {
			if c.GetHeader("HX-Request") == "true" {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}

		// Validate and reorganize the graph layout
		validateGraph(&graph)
		removeIsolatedNodes(&graph)
		reorganizeGraphLayout(&graph)

		// If this is an HTMX request
		if c.GetHeader("HX-Request") == "true" {
			// Serialize graph data to pass to the template
			graphJSON, _ := json.Marshal(graph)

			// Return the HTML with embedded graph data
			c.HTML(http.StatusOK, "graph_container.html", gin.H{
				"GraphData": string(graphJSON),
			})
		} else {
			// Regular JSON response for non-HTMX requests
			c.JSON(http.StatusOK, graph)
		}
	})
}

// Add this to your main() function
func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	// Setup existing endpoints
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Setup HTMX-specific endpoints
	setupHtmxEndpoints(r)

	r.Run(":8080")
}
