package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"finance-tracker/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
)

// AuthService предоставляет методы для регистрации и аутентификации пользователей
type AuthService struct {
	userRepo UserRepository
	jwtSecret []byte
	jwtTTL    time.Duration
}

// UserRepository определяет интерфейс для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.CreateUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (repository.User, error)
}

// CreateUserParams параметры для создания пользователя
type CreateUserParams struct {
	Email        string
	PasswordHash string
}

// CreateUserRow результат создания пользователя
type CreateUserRow struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}

// User модель пользователя
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// AuthTokens содержит пару токенов
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// RegisterInput параметры регистрации
type RegisterInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginInput параметры входа
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// NewAuthService создает новый сервис аутентификации
func NewAuthService(userRepo UserRepository, jwtSecret string, jwtTTL time.Duration) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		jwtTTL:    jwtTTL,
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthTokens, error) {
	// Проверяем, существует ли пользователь с таким email
	_, err := s.userRepo.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return AuthTokens{}, ErrUserAlreadyExists
	}
	// Если ошибка не nil, проверяем, это ли "user not found"
	// В данном случае просто игнорируем, так как нам нужно, чтобы пользователя не существовало

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthTokens{}, errors.New("failed to hash password")
	}

	// Создаем пользователя
	user, err := s.userRepo.CreateUser(ctx, repository.CreateUserParams{
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		return AuthTokens{}, errors.New("failed to create user")
	}

	// Генерируем токены
	return s.generateTokens(user.ID, user.Email)
}

// Login аутентифицирует пользователя и возвращает токены
func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthTokens, error) {
	// Получаем пользователя по email
	user, err := s.userRepo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	// Сравниваем хеш пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return AuthTokens{}, ErrInvalidCredentials
	}

	// Генерируем токены
	return s.generateTokens(user.ID, user.Email)
}

// generateTokens генерирует пару JWT токенов
func (s *AuthService) generateTokens(userID uuid.UUID, email string) (AuthTokens, error) {
	now := time.Now()
	expiresAt := now.Add(s.jwtTTL)

	// Создаем claims
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"exp":     expiresAt.Unix(),
		"iat":     now.Unix(),
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	accessToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return AuthTokens{}, errors.New("failed to sign token")
	}

	// Для refresh токена используем тот же подход (в продакшене лучше использовать отдельный секрет и TTL)
	refreshClaims := jwt.MapClaims{
		"user_id": userID.String(),
		"type":    "refresh",
		"exp":     now.Add(24 * time.Hour).Unix(), // Refresh токен живет дольше
		"iat":     now.Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return AuthTokens{}, errors.New("failed to sign refresh token")
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}

// ValidateToken проверяет валидность JWT токена и возвращает userID
func (s *AuthService) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, errors.New("user_id not found in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid user_id in token")
	}

	return userID, nil
}
