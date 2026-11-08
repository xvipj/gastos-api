package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
	"github.com/xavie/gastos-api/internal/services"
)

func TestAuthServiceRefreshRotationAndReuse(t *testing.T) {
	db, cfg, _ := newServiceTestEnv(t)
	authService := services.NewAuthService(
		repositories.NewUserRepository(db),
		repositories.NewRefreshTokenRepository(db),
		cfg,
	)

	registered, err := authService.Register(context.Background(), dto.RegisterRequest{
		Name:     "Ana",
		Email:    "ANA@example.com",
		Password: "super-secret-123",
	})
	require.NoError(t, err)
	require.Equal(t, "ana@example.com", registered.User.Email)

	// El refresh token se guarda hasheado, nunca en claro.
	var stored models.RefreshToken
	require.NoError(t, db.First(&stored).Error)
	require.NotEqual(t, registered.RefreshToken, stored.TokenHash)
	require.Len(t, stored.TokenHash, 64)

	userID, err := authService.ValidateAccessToken(registered.AccessToken)
	require.NoError(t, err)
	require.Equal(t, registered.User.ID, userID)

	refreshed, err := authService.Refresh(context.Background(), registered.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, registered.RefreshToken, refreshed.RefreshToken)
	require.NotEqual(t, registered.AccessToken, refreshed.AccessToken)

	// Reutilizar el token anterior responde 401.
	_, err = authService.Refresh(context.Background(), registered.RefreshToken)
	require.ErrorIs(t, err, services.ErrUnauthorized)

	// La reutilización revoca todas las sesiones, incluida la recién creada.
	_, err = authService.Refresh(context.Background(), refreshed.RefreshToken)
	require.ErrorIs(t, err, services.ErrUnauthorized)

	_, err = authService.ValidateAccessToken("token-invalido")
	require.ErrorIs(t, err, services.ErrUnauthorized)
}

func TestAuthServiceLogoutLoginAndExpiration(t *testing.T) {
	db, cfg, _ := newServiceTestEnv(t)
	authService := services.NewAuthService(
		repositories.NewUserRepository(db),
		repositories.NewRefreshTokenRepository(db),
		cfg,
	)

	registered, err := authService.Register(context.Background(), dto.RegisterRequest{
		Name:     "Luis",
		Email:    "luis@example.com",
		Password: "super-secret-123",
	})
	require.NoError(t, err)

	require.NoError(t, authService.Logout(context.Background(), registered.RefreshToken))
	_, err = authService.Refresh(context.Background(), registered.RefreshToken)
	require.ErrorIs(t, err, services.ErrUnauthorized)
	require.NoError(t, authService.Logout(context.Background(), registered.RefreshToken))

	_, err = authService.Login(context.Background(), dto.LoginRequest{
		Email:    "luis@example.com",
		Password: "incorrecta",
	})
	require.ErrorIs(t, err, services.ErrUnauthorized)

	loggedIn, err := authService.Login(context.Background(), dto.LoginRequest{
		Email:    "luis@example.com",
		Password: "super-secret-123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, loggedIn.AccessToken)
	require.NotEmpty(t, loggedIn.RefreshToken)

	// Un refresh token expirado no puede usarse.
	require.NoError(t, db.Model(&models.RefreshToken{}).
		Where("user_id = ?", registered.User.ID).
		Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error)
	_, err = authService.Refresh(context.Background(), loggedIn.RefreshToken)
	require.ErrorIs(t, err, services.ErrUnauthorized)
}
