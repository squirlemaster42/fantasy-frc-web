package view

import (
	"strings"

	"golang.org/x/net/html"
)

// findElementByAttr recursively searches the HTML node tree for the first element
// with the given tag name and attribute value.
func findElementByAttr(n *html.Node, tagName, attrName, attrValue string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tagName {
		for _, attr := range n.Attr {
			if attr.Key == attrName && attr.Val == attrValue {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElementByAttr(c, tagName, attrName, attrValue); found != nil {
			return found
		}
	}
	return nil
}

// findElementByTag recursively searches for the first element with the given tag name.
func findElementByTag(n *html.Node, tagName string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tagName {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElementByTag(c, tagName); found != nil {
			return found
		}
	}
	return nil
}

// findAllElementsByTag recursively collects all elements with the given tag name.
func findAllElementsByTag(n *html.Node, tagName string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tagName {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllElementsByTag(c, tagName)...)
	}
	return result
}

// textContent returns the concatenated text content of a node and its descendants.
func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}

// hasAttrContaining checks if the given node has an attribute whose value contains the substring.
func hasAttrContaining(n *html.Node, name, substring string) bool {
	if n == nil {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == name && strings.Contains(attr.Val, substring) {
			return true
		}
	}
	return false
}

// getAttr returns the value of an attribute on the node, or empty string if absent.
func getAttr(n *html.Node, name string) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}


