package view

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestLandingPage(t *testing.T) {
	var buf strings.Builder
	err := Landing(false, "").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()

	// Parse for deeper assertions
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("doctype and html structure", func(t *testing.T) {
		assert.Contains(t, htmlStr, "<!doctype html>")
		assert.Contains(t, htmlStr, `<html lang="en" data-theme="fantasy-frc">`)
	})

	t.Run("htmx and alpine scripts", func(t *testing.T) {
		assert.Contains(t, htmlStr, `/js/htmx.min.js`)
		assert.Contains(t, htmlStr, `/js/alpine.min.js`)
		assert.Contains(t, htmlStr, `/js/app.js`)
		assert.Contains(t, htmlStr, `defer src="/js/alpine.min.js"`)
	})

	t.Run("stylesheet and favicon", func(t *testing.T) {
		assert.Contains(t, htmlStr, `/css/styles.css`)
		assert.Contains(t, htmlStr, `href="/img/favicon.svg"`)
	})

	t.Run("hx-boost on body", func(t *testing.T) {
		assert.Contains(t, htmlStr, `hx-boost="true"`)
	})

	t.Run("meta tags", func(t *testing.T) {
		assert.Contains(t, htmlStr, `charset="UTF-8"`)
		assert.Contains(t, htmlStr, `name="viewport"`)
		assert.Contains(t, htmlStr, `content="width=device-width, initial-scale=1.0"`)
		assert.Contains(t, htmlStr, `google`)
		assert.Contains(t, htmlStr, `notranslate`)
	})

	t.Run("title contains Fantasy FRC", func(t *testing.T) {
		title := findElementByTag(doc, "title")
		require.NotNil(t, title)
		assert.Contains(t, textContent(title), "Fantasy FRC")
	})

	t.Run("navigation links present", func(t *testing.T) {
		loginLink := findElementByAttr(doc, "a", "href", "/login")
		require.NotNil(t, loginLink)
		assert.Contains(t, textContent(loginLink), "Log In")

		registerLink := findElementByAttr(doc, "a", "href", "/register")
		require.NotNil(t, registerLink)
		assert.Contains(t, textContent(registerLink), "Sign Up")
	})

	t.Run("hero CTA buttons", func(t *testing.T) {
		// Hero register CTA
		heroRegister := findElementByAttr(doc, "a", "href", "/register")
		require.NotNil(t, heroRegister)
		// There are multiple /register links; check the first one has CTA text
		// We use the DOM helper to look for the first one with specific text
		found := false
		for _, a := range findAllElementsByTag(doc, "a") {
			if getAttr(a, "href") == "/register" && strings.Contains(textContent(a), "Create Free Account") {
				found = true
				break
			}
		}
		assert.True(t, found, "hero CTA 'Create Free Account' not found")
	})

	t.Run("how it works section", func(t *testing.T) {
		assert.Contains(t, htmlStr, "How It Works")
		assert.Contains(t, htmlStr, "Create a Draft")
		assert.Contains(t, htmlStr, "Draft Teams")
		assert.Contains(t, htmlStr, "Score Points")
	})

	t.Run("footer content", func(t *testing.T) {
		assert.Contains(t, htmlStr, "Powered by")
		assert.Contains(t, htmlStr, "The Blue Alliance")
	})
}

func TestLandingPage_LoggedIn(t *testing.T) {
	var buf strings.Builder
	err := Landing(true, "TestUser").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("shows shared navbar when logged in", func(t *testing.T) {
		// The shared navbar links the logo to /u/home
		homeLink := findElementByAttr(doc, "a", "href", "/u/home")
		require.NotNil(t, homeLink)
		assert.Contains(t, textContent(homeLink), "Fantasy FRC")
	})

	t.Run("shows username in navbar", func(t *testing.T) {
		assert.Contains(t, htmlStr, "TestUser")
	})
}

func TestLandingPage_LoggedOut(t *testing.T) {
	var buf strings.Builder
	err := Landing(false, "").Render(context.Background(), &buf)
	require.NoError(t, err)

	htmlStr := buf.String()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	require.NoError(t, err)

	t.Run("shows logged out marketing nav", func(t *testing.T) {
		loginLink := findElementByAttr(doc, "a", "href", "/login")
		require.NotNil(t, loginLink)
		assert.Contains(t, textContent(loginLink), "Log In")

		registerLink := findElementByAttr(doc, "a", "href", "/register")
		require.NotNil(t, registerLink)
		assert.Contains(t, textContent(registerLink), "Sign Up")
	})

	t.Run("fantasy frc logo links to landing page", func(t *testing.T) {
		homeLink := findElementByAttr(doc, "a", "href", "/")
		require.NotNil(t, homeLink)
		assert.Contains(t, textContent(homeLink), "Fantasy FRC")
	})
}
