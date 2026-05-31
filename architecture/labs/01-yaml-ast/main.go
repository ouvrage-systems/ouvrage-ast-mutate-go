package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// This lab demonstrates how yaml.v3 parses a YAML document into an
// Abstract Syntax Tree (AST) using the yaml.Node structure, and how we
// can traverse and modify this tree programmatically while preserving
// all comments and formatting.

const sampleYAML = `
# Global configuration file for Ouvrage Proxy
version: "v1.2.0"
metadata:
  name: "auth-proxy" # Unique identifier
  labels:
    env: "production"
    tier: "frontend"

# List of upstream target endpoints
upstreams:
  - host: "10.0.0.5"
    port: 8080 # Primary instance
  - host: "10.0.0.6"
    port: 8080 # Fallback instance
`

func main() {
	fmt.Println("=== LAB 01: YAML AST Exploration ===")

	// 1. Parsing the raw YAML into a yaml.Node AST.
	// In yaml.v3, unmarshalling into a yaml.Node returns the root of the AST.
	var root yaml.Node
	err := yaml.Unmarshal([]byte(sampleYAML), &root)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// 2. Printing the AST tree representation
	fmt.Println("\n--- AST Tree Structure (Visualized) ---")
	printNodeTree(&root, 0)

	// 3. Mutating the AST programmatically
	// Let's modify the name under metadata.name from "auth-proxy" to "auth-proxy-v2".
	fmt.Println("\n--- Mutating 'metadata.name' to 'auth-proxy-v2' ---")
	
	// We locate the metadata mapping node, then locate the name key, and update its value node.
	err = mutateMetadataName(&root, "auth-proxy-v2")
	if err != nil {
		log.Fatalf("Mutation failed: %v", err)
	}
	fmt.Println("Mutation successful!")

	// 4. Marshaling the mutated AST back to YAML
	fmt.Println("\n--- Mutated YAML Output (Preserving Comments) ---")
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	err = encoder.Encode(&root)
	if err != nil {
		log.Fatalf("Failed to encode YAML: %v", err)
	}
}

// printNodeTree recursively walks the yaml.Node AST and prints a tree diagram.
func printNodeTree(node *yaml.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	kindStr := nodeKindToString(node.Kind)

	// Build a descriptive label for the node
	var details []string
	if node.Value != "" {
		details = append(details, fmt.Sprintf("value: %q", node.Value))
	}
	if node.Tag != "" {
		details = append(details, fmt.Sprintf("tag: %q", node.Tag))
	}
	if node.HeadComment != "" {
		details = append(details, fmt.Sprintf("headComment: %q", strings.TrimSpace(node.HeadComment)))
	}
	if node.LineComment != "" {
		details = append(details, fmt.Sprintf("lineComment: %q", strings.TrimSpace(node.LineComment)))
	}

	detailsStr := ""
	if len(details) > 0 {
		detailsStr = " (" + strings.Join(details, ", ") + ")"
	}

	fmt.Printf("%s- [%s]%s at Line %d\n", indent, kindStr, detailsStr, node.Line)

	for _, child := range node.Content {
		printNodeTree(child, depth+1)
	}
}

// mutateMetadataName traverses the AST to find metadata -> name and modifies its value.
// In yaml.v3:
// - A DocumentNode has 1 child (usually the root MappingNode representing the top-level keys).
// - A MappingNode's Content is a flat slice where odd elements are keys (ScalarNode) and even elements are values.
func mutateMetadataName(root *yaml.Node, newName string) error {
	if root.Kind != yaml.DocumentNode {
		return fmt.Errorf("expected root to be DocumentNode, got %v", root.Kind)
	}

	if len(root.Content) == 0 {
		return fmt.Errorf("empty document node")
	}

	topMapping := root.Content[0]
	if topMapping.Kind != yaml.MappingNode {
		return fmt.Errorf("expected top level to be MappingNode, got %v", topMapping.Kind)
	}

	// 1. Find the "metadata" key in the top level mapping
	metadataNode := findKeyInMapping(topMapping, "metadata")
	if metadataNode == nil {
		return fmt.Errorf("key 'metadata' not found")
	}

	if metadataNode.Kind != yaml.MappingNode {
		return fmt.Errorf("expected 'metadata' to be a MappingNode, got %v", metadataNode.Kind)
	}

	// 2. Find the "name" key inside the "metadata" mapping
	nameNode := findKeyInMapping(metadataNode, "name")
	if nameNode == nil {
		return fmt.Errorf("key 'metadata.name' not found")
	}

	if nameNode.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected 'name' value to be a ScalarNode, got %v", nameNode.Kind)
	}

	// 3. Perform the mutation
	nameNode.Value = newName

	return nil
}

// findKeyInMapping searches a MappingNode's children for a key matching targetKey,
// and returns the corresponding value node (the next element in the Content slice).
func findKeyInMapping(mapping *yaml.Node, targetKey string) *yaml.Node {
	// Content is ordered as: [Key1, Value1, Key2, Value2, ...]
	for i := 0; i < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		if keyNode.Value == targetKey {
			// The value node is immediately following the key node
			return mapping.Content[i+1]
		}
	}
	return nil
}

// nodeKindToString converts the numeric Kind of a yaml.Node to a human-readable string.
func nodeKindToString(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "Document"
	case yaml.SequenceNode:
		return "Sequence (List)"
	case yaml.MappingNode:
		return "Mapping (Map)"
	case yaml.ScalarNode:
		return "Scalar (Value)"
	case yaml.AliasNode:
		return "Alias"
	default:
		return "Unknown"
	}
}
