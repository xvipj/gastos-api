package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
)

const jwtIssuer = "gastos-api"

// accessClaims son los claims del access token JWT.
type accessClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthService implementa registro, login, rotación y revocación de tokens.
type AuthService struct {
	users         *repositories.UserRepository
	refreshTokens *repositories.RefreshTokenRepository
	cfg           *config.Config
}

// NewAuthService construye el servicio de autenticación.
func NewAuthService(
	users *repositories.UserRepository,
	refreshTokens *repositories.RefreshTokenRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, cfg: cfg}
}

// Register crea un usuario y devuelve su primer par de tokens.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := normalizeEmail(req.Email)

	if len([]byte(req.Password)) > 72 {
		return nil, NewValidationError("la contraseña no puede superar los 72 bytes")
	}

	_, err := s.users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		return nil, NewConflictError("el email ya está registrado")
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, fmt.Errorf("comprobando email existente: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generando hash de contraseña: %w", err)
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, NewConflictError("el email ya está registrado")
		}
		return nil, fmt.Errorf("creando usuario: %w", err)
	}

	response, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Login valida las credenciales y devuelve un nuevo par de tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	email := normalizeEmail(req.Email)

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUnauthorizedError("credenciales inválidas")
		}
		return nil, fmt.Errorf("buscando usuario para login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, NewUnauthorizedError("credenciales inválidas")
	}

	return s.issueTokens(ctx, user)
}

// Refresh valida y rota un refresh token. Un token ya revocado provoca 401 y,
// por seguridad, la revocación de todas las sesiones del usuario.
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*dto.AuthResponse, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, NewValidationError("el refresh token es obligatorio")
	}

	token, err := s.refreshTokens.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUnauthorizedError("refresh token inválido")
		}
		return nil, fmt.Errorf("buscando refresh token: %w", err)
	}

	if token.RevokedAt != nil {
		if revokeErr := s.refreshTokens.RevokeAllForUser(ctx, token.UserID); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, NewUnauthorizedError("refresh token revocado")
	}

	if !time.Now().UTC().Before(token.ExpiresAt) {
		return nil, NewUnauthorizedError("refresh token expirado")
	}

	user, err := s.users.FindByID(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUnauthorizedError("usuario no encontrado")
		}
		return nil, fmt.Errorf("buscando usuario del refresh token: %w", err)
	}

	now := time.Now().UTC()
	accessToken, expiresIn, err := s.buildAccessToken(user.ID, now)
	if err != nil {
		return nil, err
	}
	newRaw, newRecord, err := s.newRefreshToken(user.ID, now)
	if err != nil {
		return nil, err
	}

	if err := s.refreshTokens.Rotate(ctx, token.ID, newRecord); err != nil {
		if errors.Is(err, repositories.ErrTokenAlreadyRevoked) {
			_ = s.refreshTokens.RevokeAllForUser(ctx, token.UserID)
			return nil, NewUnauthorizedError("refresh token revocado")
		}
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User:         userResponse(user),
	}, nil
}

// Logout revoca el refresh token recibido. Es idempotente: un token ya
// revocado o desconocido no produce error.
func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return NewValidationError("el refresh token es obligatorio")
	}

	token, err := s.refreshTokens.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("buscando refresh token para logout: %w", err)
	}
	if token.RevokedAt != nil {
		return nil
	}
	return s.refreshTokens.Revoke(ctx, token.ID)
}

// ValidateAccessToken comprueba la firma y la expiración de un access token.
func (s *AuthService) ValidateAccessToken(rawToken string) (uint, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("método de firma no permitido: %s", t.Method.Alg())
			}
			return []byte(s.cfg.JWTSecret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(jwtIssuer),
	)
	if err != nil || !token.Valid || claims.UserID == 0 {
		return 0, NewUnauthorizedError("access token inválido o expirado")
	}
	return claims.UserID, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*dto.AuthResponse, error) {
	now := time.Now().UTC()

	accessToken, expiresIn, err := s.buildAccessToken(user.ID, now)
	if err != nil {
		return nil, err
	}

	rawRefresh, record, err := s.newRefreshToken(user.ID, now)
	if err != nil {
		return nil, err
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User:         userResponse(user),
	}, nil
}

func (s *AuthService) buildAccessToken(userID uint, now time.Time) (string, int64, error) {
	tokenID, err := randomHex(16)
	if err != nil {
		return "", 0, err
	}

	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, fmt.Errorf("firmando access token: %w", err)
	}
	return signed, int64(s.cfg.AccessTokenTTL.Seconds()), nil
}

func (s *AuthService) newRefreshToken(userID uint, now time.Time) (string, *models.RefreshToken, error) {
	raw, err := randomHex(32)
	if err != nil {
		return "", nil, fmt.Errorf("generando refresh token: %w", err)
	}
	return raw, &models.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
	}, nil
}

func randomHex(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generando bytes aleatorios: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func userResponse(user *models.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
