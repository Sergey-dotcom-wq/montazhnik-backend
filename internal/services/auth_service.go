package services

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// AuthServiceImpl реализация AuthService
type AuthServiceImpl struct {
	userRepo  domain.UserRepository
	orgRepo   domain.OrganizationRepository
	jwtSecret string
}

// NewAuthService конструктор
func NewAuthService(userRepo domain.UserRepository, orgRepo domain.OrganizationRepository, jwtSecret string) *AuthServiceImpl {
	return &AuthServiceImpl{userRepo: userRepo, orgRepo: orgRepo, jwtSecret: jwtSecret}
}

// Register создаёт пользователя.
// Идентификатор — username (≤10 символов, латиница+цифры), уникальный.
// orgName — имя коллектива: если существует — присоединение,
// если нет — создаётся новый коллектив, пользователь становится его admin.
// phone — необязательное поле профиля (НЕ идентификатор, в аудит не попадает).
// referralCode — код партнёра: привязывает пользователя к партнёру
// для начисления выплаты при активации подписки.
func (s *AuthServiceImpl) Register(ctx context.Context, fullName, username, password, role, orgName, phone, email, referralCode string) (*domain.User, error) {
	existing, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, errors.New("username already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		FullName:     fullName,
		Username:     username,
		Phone:        phone,
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	user, err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// Коллектив: присоединение к существующему или создание нового.
	orgID, isNew, oerr := s.resolveOrg(ctx, orgName, user.ID)
	if oerr != nil {
		return nil, oerr
	}
	if isNew {
		// Создатель коллектива становится его admin — сохраняем в БД.
		user.Role = "admin"
		_ = s.userRepo.SetRole(ctx, user.ID, "admin")
	}
	user.OrgID = orgID
	_ = s.userRepo.SetOrg(ctx, user.ID, orgID)

	// Если при регистрации указан реферальный код — привязываем пользователя
	// к партнёру (выплата при активации подписки).
	if referralCode != "" {
		partnerID, perr := s.userRepo.PartnerIDByReferralCode(ctx, referralCode)
		if perr == nil {
			_ = s.userRepo.SetReferralPartner(ctx, user.ID, partnerID)
		}
	}

	return user, nil
}

// AnonymizeAccount удаляет персональные данные пользователя при выходе
// из приложения (требование 152-ФЗ). В истории/аудите остаётся только
// username и UUID — они не являются персональными данными в смысле 152-ФЗ.
func (s *AuthServiceImpl) AnonymizeAccount(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.AnonymizeAccount(ctx, userID)
}

// resolveOrg определяет org_id для нового пользователя.
// orgName пуст — организация по умолчанию (обратная совместимость).
// Имя существует — присоединяемся; иначе создаём коллектив (пользователь admin).
func (s *AuthServiceImpl) resolveOrg(ctx context.Context, orgName string, userID uuid.UUID) (uuid.UUID, bool, error) {
	const defaultOrg = "11111111-1111-1111-1111-111111111111"

	if orgName == "" {
		return uuid.MustParse(defaultOrg), false, nil
	}

	existingID, err := s.orgRepo.FindByName(ctx, orgName)
	if err == nil && existingID != uuid.Nil {
		return existingID, false, nil
	}

	invite := genInviteCode()
	newID, cerr := s.orgRepo.Create(ctx, orgName, userID, invite)
	if cerr != nil {
		return uuid.Nil, false, errors.New("failed to create organization")
	}
	return newID, true, nil
}

// genInviteCode генерирует короткий код приглашения коллектива.
func genInviteCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

// BindHwID привязывает устройство к пользователю (обновляет или заполняет hw_id).
func (s *AuthServiceImpl) BindHwID(ctx context.Context, userID uuid.UUID, hwID string) {
	_ = s.userRepo.BindHwID(ctx, userID, hwID)
}

// Login выполняет вход по username (единственный идентификатор).
func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, *domain.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		log.Printf("LOGIN: GetByUsername(%q) err %v", username, err)
		return "", nil, errors.New("invalid username or password")
	}
	log.Printf("LOGIN: %q найден, hash=%q", username, user.PasswordHash)

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Printf("LOGIN: bcrypt mismatch for %q, input len=%d, err %v", username, len(password), err)
		return "", nil, errors.New("invalid username or password")
	}
	log.Printf("LOGIN: bcrypt OK for %q", username)

	orgID := s.userRepo.GetOrgID(ctx, user.ID)
	orgIDStr := ""
	if orgID != uuid.Nil {
		orgIDStr = orgID.String()
	}

	return s.issueToken(user, orgIDStr), user, nil
}

// IssueToken генерирует JWT для пользователя (используется после MFA-верификации).
func (s *AuthServiceImpl) IssueToken(userID uuid.UUID) (string, *domain.User, error) {
	user, err := s.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		return "", nil, err
	}
	orgID := s.userRepo.GetOrgID(context.Background(), userID)
	orgIDStr := ""
	if orgID != uuid.Nil {
		orgIDStr = orgID.String()
	}
	token := s.issueToken(user, orgIDStr)
	if token == "" {
		return "", nil, errors.New("token generation failed")
	}
	return token, user, nil
}

// issueToken генерирует JWT для пользователя.
func (s *AuthServiceImpl) issueToken(user *domain.User, orgIDStr string) string {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"user": user.Username,
		"role": user.Role,
		"org":  orgIDStr,
		"exp":  time.Now().Add(72 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return ""
	}
	return tokenString
}

// ValidateToken проверяет JWT-токен и возвращает ID пользователя, роль и org_id.
func (s *AuthServiceImpl) ValidateToken(tokenString string) (uuid.UUID, string, uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, "", uuid.Nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", uuid.Nil, errors.New("invalid claims")
	}

	sub, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	orgStr, _ := claims["org"].(string)
	userID, _ := uuid.Parse(sub)
	orgID, _ := uuid.Parse(orgStr)

	return userID, role, orgID, nil
}