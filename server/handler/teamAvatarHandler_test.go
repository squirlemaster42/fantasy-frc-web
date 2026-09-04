package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetTeamAvatar(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/team/254/avatar", "", "")
		c.SetParamNames("id")
		c.SetParamValues("254")

		avatarBytes := []byte("fake-png-bytes")
		mockStore := &mockAvatarStore{avatar: avatarBytes}

		h := &Handler{
			Services: ServiceGroup{
				AvatarStore: mockStore,
			},
		}

		err := h.GetTeamAvatar(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "image/png", rec.Header().Get(echo.HeaderContentType))
		assert.Contains(t, rec.Header().Get("Cache-Control"), "private, max-age=")
		assert.Equal(t, avatarBytes, rec.Body.Bytes())
	})

	t.Run("invalid team number", func(t *testing.T) {
		e, c, rec := setupTestContext(t, http.MethodGet, "/u/team/abc/avatar", "", "")
		c.SetParamNames("id")
		c.SetParamValues("abc")

		h := &Handler{
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{},
			},
		}

		err := h.GetTeamAvatar(c)
		if assert.Error(t, err) {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("avatar store error", func(t *testing.T) {
		e, c, rec := setupTestContext(t, http.MethodGet, "/u/team/254/avatar", "", "")
		c.SetParamNames("id")
		c.SetParamValues("254")

		mockStore := &mockAvatarStore{err: errors.New("redis unavailable")}

		h := &Handler{
			Services: ServiceGroup{
				AvatarStore: mockStore,
			},
		}

		err := h.GetTeamAvatar(c)
		if assert.Error(t, err) {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
