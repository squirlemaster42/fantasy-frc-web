package login

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"server/types"
	"server/view"
)

// Helper functions (copied from view/testutil_test.go to avoid cross-package test helpers)

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

func hasAttr(n *html.Node, name, value string) bool {
	if n == nil {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == name && attr.Val == value {
			return true
		}
	}
	return false
}

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

func TestLoginForm(t *testing.T) {
	var buf strings.Builder
	err := LoginIndex(false, "", 8, "test-csrf-token").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("htmx form attributes", func(t *testing.T) {
		form := findElementByTag(doc, "form")
		require.NotNil(t, form)
		assert.True(t, hasAttr(form, "hx-post", "/login"), "form should have hx-post=/login")
		assert.True(t, hasAttr(form, "hx-target", "#login-box"), "form should have hx-target=#login-box")
		assert.Contains(t, getAttr(form, "hx-swap"), "outerHTML", "form should use outerHTML swap")
	})

	t.Run("username input", func(t *testing.T) {
		inputs := findAllElementsByTag(doc, "input")
		var usernameInput *html.Node
		for _, input := range inputs {
			if getAttr(input, "name") == "username" {
				usernameInput = input
				break
			}
		}
		require.NotNil(t, usernameInput, "username input not found")
		assert.Equal(t, "text", getAttr(usernameInput, "type"))
		assert.True(t, hasAttr(usernameInput, "required", "required") || hasAttr(usernameInput, "required", ""), "username should be required")
		assert.True(t, hasAttr(usernameInput, "autofocus", "autofocus") || hasAttr(usernameInput, "autofocus", ""), "username should have autofocus")
	})

	t.Run("password input", func(t *testing.T) {
		inputs := findAllElementsByTag(doc, "input")
		var passwordInput *html.Node
		for _, input := range inputs {
			if getAttr(input, "name") == "password" {
				passwordInput = input
				break
			}
		}
		require.NotNil(t, passwordInput, "password input not found")
		assert.Equal(t, "password", getAttr(passwordInput, "type"))
		assert.True(t, hasAttr(passwordInput, "required", "required") || hasAttr(passwordInput, "required", ""), "password should be required")
		assert.Contains(t, getAttr(passwordInput, "minlength"), "8", "password should have minlength=8")
	})

	t.Run("csrf token input", func(t *testing.T) {
		inputs := findAllElementsByTag(doc, "input")
		var csrfInput *html.Node
		for _, input := range inputs {
			if getAttr(input, "name") == "csrf_token" {
				csrfInput = input
				break
			}
		}
		require.NotNil(t, csrfInput, "csrf_token input not found")
		assert.Equal(t, "hidden", getAttr(csrfInput, "type"))
		assert.Equal(t, "test-csrf-token", getAttr(csrfInput, "value"))
	})

	t.Run("submit button", func(t *testing.T) {
		button := findElementByTag(doc, "button")
		require.NotNil(t, button)
		assert.Equal(t, "submit", getAttr(button, "type"))
		assert.Contains(t, textContent(button), "Sign In")
	})

	t.Run("register link", func(t *testing.T) {
		registerLink := findElementByAttr(doc, "a", "href", "/register")
		require.NotNil(t, registerLink)
		assert.Contains(t, textContent(registerLink), "Create one here")
	})

	t.Run("alpine loading state", func(t *testing.T) {
		assert.Contains(t, htmlStr, `x-data="{ loading: false }"`)
		assert.Contains(t, htmlStr, `@htmx:before-request`)
		assert.Contains(t, htmlStr, `@htmx:after-request`)
		assert.Contains(t, htmlStr, `x-show="!loading"`)
		assert.Contains(t, htmlStr, `x-show="loading"`)
		assert.Contains(t, htmlStr, `x-cloak`)
	})

	t.Run("error message renders", func(t *testing.T) {
		var errBuf strings.Builder
		err := LoginIndex(false, "Invalid username or password", 8, "csrf").Render(context.Background(), &errBuf)
		require.NoError(t, err)
		assert.Contains(t, errBuf.String(), "Invalid username or password")
	})

	t.Run("disabled state when fromProtected", func(t *testing.T) {
		var disabledBuf strings.Builder
		err := LoginIndex(true, "", 8, "csrf").Render(context.Background(), &disabledBuf)
		require.NoError(t, err)
		assert.Contains(t, disabledBuf.String(), `disabled`)
	})
}

func TestLoginPage(t *testing.T) {
	var buf strings.Builder
	// Login wraps the login component in the Index layout
	// We just verify it renders without error and contains the expected structure
	err := Login("Sign In", false, LoginIndex(false, "", 8, "csrf")).Render(t.Context(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	assert.Contains(t, htmlStr, "<!doctype html>")
	assert.Contains(t, htmlStr, `<html lang="en" data-theme="fantasy-frc">`)
	assert.Contains(t, htmlStr, "Sign In")
	assert.Contains(t, htmlStr, `hx-boost="true"`)
}

// Test that the HTML helpers from the view package are available
func TestLoginUsesViewHelpers(t *testing.T) {
	// This test ensures the login package compiles with view package imports
	_ = view.Index("Sign In", false, "", types.NewPageData(0, "", false))
}
