package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type TeamListNode struct {
	Name   string  `json:"name"`
	Parent *string `json:"parent"`
}

type TeamTreeNode struct {
	Name     string
	Children []*TeamTreeNode
}

type ParseResult struct {
	Org    string
	Format string
	Help   bool
}

func load(org string) ([]TeamListNode, error) {
	cmd := exec.Command(
		"gh",
		"api",
		fmt.Sprintf("/orgs/%s/teams", org),
		"--paginate",
		"--jq",
		".[] | {name: .name, parent: .parent.name}",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(msg, "HTTP 404") {
			return nil, fmt.Errorf(
				"error: %s\nhint: ensure %q is an organization login (not owner/repo) and your auth has read:org (`gh auth refresh -h github.com -s read:org`)",
				msg,
				org,
			)
		}
		return nil, fmt.Errorf("error: %s", msg)
	}

	nodes := make([]TeamListNode, 0)
	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var node TeamListNode
		if err := json.Unmarshal([]byte(line), &node); err != nil {
			return nil, fmt.Errorf("error: failed to parse gh api output line %q: %w", line, err)
		}
		nodes = append(nodes, node)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error: failed to read gh api output: %w", err)
	}

	return nodes, nil
}

func structure(org string, nodes []TeamListNode) *TeamTreeNode {
	treeNodeMap := make(map[string]*TeamTreeNode, len(nodes))

	for _, node := range nodes {
		treeNodeMap[node.Name] = &TeamTreeNode{
			Name:     node.Name,
			Children: make([]*TeamTreeNode, 0),
		}
	}

	root := &TeamTreeNode{
		Name:     org,
		Children: make([]*TeamTreeNode, 0),
	}

	for _, listNode := range nodes {
		treeNode := treeNodeMap[listNode.Name]
		if treeNode == nil {
			continue
		}

		if listNode.Parent != nil && *listNode.Parent != "" {
			parent := treeNodeMap[*listNode.Parent]
			if parent != nil {
				parent.Children = append(parent.Children, treeNode)
				continue
			}
		}
		root.Children = append(root.Children, treeNode)
	}

	return sortTree(root)
}

func collectPaths(node *TeamTreeNode, prefix string) []string {
	path := node.Name
	if prefix != "" {
		path = prefix + "/" + node.Name
	}

	result := []string{path}
	for _, child := range node.Children {
		result = append(result, collectPaths(child, path)...)
	}
	return result
}

func sortTree(node *TeamTreeNode) *TeamTreeNode {
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})

	for _, child := range node.Children {
		sortTree(child)
	}

	return node
}

func countNodes(node *TeamTreeNode) int {
	count := len(node.Children)
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}

func printTree(root *TeamTreeNode) {
	var printRecursive func(node *TeamTreeNode, prefix, style string)
	printRecursive = func(node *TeamTreeNode, prefix, style string) {
		label := node.Name
		if len(node.Children) > 0 {
			label += "/"
		}

		line := ""
		switch style {
		case "root":
			line = label
		case "node-middle":
			line = prefix + "├── " + label
		case "node-last":
			line = prefix + "└── " + label
		}

		fmt.Println(line)

		childPrefix := ""
		switch style {
		case "root":
			childPrefix = ""
		case "node-middle":
			childPrefix = prefix + "│   "
		case "node-last":
			childPrefix = prefix + "    "
		}

		for i, child := range node.Children {
			childStyle := "node-middle"
			if i == len(node.Children)-1 {
				childStyle = "node-last"
			}
			printRecursive(child, childPrefix, childStyle)
		}
	}

	printRecursive(root, "", "root")
	fmt.Println()
	fmt.Printf("Total teams: %d\n", countNodes(root))
}

func printList(root *TeamTreeNode) {
	paths := make([]string, 0)
	for _, node := range root.Children {
		paths = append(paths, collectPaths(node, root.Name)...)
	}

	sort.Strings(paths)
	for _, path := range paths {
		fmt.Println(path)
	}

	fmt.Println()
	fmt.Printf("Total teams: %d\n", countNodes(root))
}

func help() string {
	return `
Display GitHub organization team structure

Usage:
  gh team <organization> [options]

Arguments:
  <organization>    GitHub organization name (required)

Options:
  --format=<type>   Output format: tree (default) or list
  --help, -h        Show this help message

Examples:
  gh team your-org
  gh team your-org --format=tree
  gh team your-org --format=list
`
}

func parse(args []string) (ParseResult, error) {
	argv := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			argv = append(argv, parts[0], parts[1])
			continue
		}
		argv = append(argv, arg)
	}

	format := "tree"
	showHelp := false
	positional := make([]string, 0)

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if arg == "--help" || arg == "-h" {
			showHelp = true
			continue
		}

		if arg == "--format" {
			i++
			if i < len(argv) {
				format = argv[i]
			}
			continue
		}

		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		}
	}

	if showHelp {
		return ParseResult{Help: true}, nil
	}

	if len(positional) == 0 {
		return ParseResult{}, errors.New("organization name is required")
	}

	if format != "tree" && format != "list" {
		return ParseResult{}, fmt.Errorf("invalid format %q. use \"tree\" or \"list\"", format)
	}

	org := positional[0]
	if slash := strings.Index(org, "/"); slash > 0 {
		org = org[:slash]
	}

	return ParseResult{
		Org:    org,
		Format: format,
	}, nil
}

func main() {
	result, err := parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		fmt.Fprintln(os.Stderr, help())
		os.Exit(1)
	}

	if result.Help {
		fmt.Println(help())
		os.Exit(0)
	}

	nodes, err := load(result.Org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	root := structure(result.Org, nodes)
	switch result.Format {
	case "tree":
		printTree(root)
	case "list":
		printList(root)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", result.Format)
		os.Exit(1)
	}
}
