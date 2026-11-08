package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xavie/gastos-api/internal/dto"
)

func TestAuthRegisterLoginAndProtectedRoute(t *testing.T) {
	app := newTestApp(t)

	registered := app.register("Ana García", "ana@example.com", "super-secret-123")
	require.NotEmpty(t, registered.AccessToken)
	require.NotEmpty(t, registered.RefreshToken)
	require.Equal(t, "Bearer", registered.TokenType)
	require.Equal(t, int64(900), registered.ExpiresIn)
	require.Equal(t, "ana@example.com", registered.User.Email)

	recorder := app.do(http.MethodGet, "/api/v1/categories", nil, "")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)

	recorder = app.do(http.MethodGet, "/api/v1/categories", nil, registered.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	categories := decodeData[[]dto.CategoryResponse](t, recorder)
	require.Empty(t, categories)

	recorder = app.do(http.MethodPost, "/api/v1/auth/login", dto.LoginRequest{
		Email:    "ANA@example.com",
		Password: "super-secret-123",
	}, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	loggedIn := decodeData[authData](t, recorder)
	require.NotEmpty(t, loggedIn.AccessToken)

	recorder = app.do(http.MethodGet, "/api/v1/categories", nil, "token-invalido")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)
}

func TestAuthRegisterValidationAndDuplicate(t *testing.T) {
	app := newTestApp(t)
	app.register("Ana", "ana@example.com", "super-secret-123")

	recorder := app.do(http.MethodPost, "/api/v1/auth/register", dto.RegisterRequest{
		Name:     "Otra Ana",
		Email:    "ana@example.com",
		Password: "super-secret-123",
	}, "")
	requireAPIError(t, recorder, http.StatusConflict, dto.CodeConflict)

	recorder = app.do(http.MethodPost, "/api/v1/auth/register", dto.RegisterRequest{
		Name:     "X",
		Email:    "no-es-un-email",
		Password: "corta",
	}, "")
	requireAPIError(t, recorder, http.StatusBadRequest, dto.CodeValidation)

	recorder = app.do(http.MethodPost, "/api/v1/auth/login", dto.LoginRequest{
		Email:    "ana@example.com",
		Password: "incorrecta",
	}, "")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)
}

func TestRefreshTokenRotationAndReuse(t *testing.T) {
	app := newTestApp(t)
	registered := app.register("Ana", "rotation@example.com", "super-secret-123")

	recorder := app.do(http.MethodPost, "/api/v1/auth/refresh", dto.RefreshRequest{
		RefreshToken: registered.RefreshToken,
	}, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	refreshed := decodeData[authData](t, recorder)
	require.NotEqual(t, registered.RefreshToken, refreshed.RefreshToken)
	require.NotEmpty(t, refreshed.AccessToken)

	// Reutilizar el token rotado debe fallar.
	recorder = app.do(http.MethodPost, "/api/v1/auth/refresh", dto.RefreshRequest{
		RefreshToken: registered.RefreshToken,
	}, "")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)

	// Por seguridad, la reutilización revoca todas las sesiones del usuario.
	recorder = app.do(http.MethodPost, "/api/v1/auth/refresh", dto.RefreshRequest{
		RefreshToken: refreshed.RefreshToken,
	}, "")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	app := newTestApp(t)
	registered := app.register("Ana", "logout@example.com", "super-secret-123")

	recorder := app.do(http.MethodPost, "/api/v1/auth/logout", dto.LogoutRequest{
		RefreshToken: registered.RefreshToken,
	}, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	recorder = app.do(http.MethodPost, "/api/v1/auth/refresh", dto.RefreshRequest{
		RefreshToken: registered.RefreshToken,
	}, "")
	requireAPIError(t, recorder, http.StatusUnauthorized, dto.CodeUnauthorized)

	// Logout es idempotente.
	recorder = app.do(http.MethodPost, "/api/v1/auth/logout", dto.LogoutRequest{
		RefreshToken: registered.RefreshToken,
	}, "")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}
