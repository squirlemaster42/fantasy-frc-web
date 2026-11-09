package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyComponent() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := w.Write([]byte("<div>ok</div>")); err != nil {
			return errors.New("failed to write component")
		}
		return nil
	})
}

func failingComponent() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return errors.New("render failed")
	})
}

func TestRenderOrLog_Success(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := RenderOrLog(c, emptyComponent(), "should not log")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "<div>ok</div>", rec.Body.String())
}

func TestRenderOrLog_Failure(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := RenderOrLog(c, failingComponent(), "render failed")

	require.Error(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestRedirect(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := Redirect(c, "/login")

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestHXRedirect(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := HXRedirect(c, "/u/home")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/u/home", rec.Header().Get("HX-Redirect"))
}

func TestBadRequest(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := BadRequest(c, "log message", "user message")

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "user message", rec.Body.String())
}

func TestForbidden(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := Forbidden(c, "log message", "user message")

	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "user message", rec.Body.String())
}

func TestUnauthorized(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := Unauthorized(c, "log message", "user message")

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "user message", rec.Body.String())
}

func TestNotFound(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := NotFound(c, "log message", "user message")

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "user message", rec.Body.String())
}

func TestInternalServerError(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "")

	err := InternalServerError(c, "log message", errors.New("boom"))

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "An error occurred", rec.Body.String())
}
