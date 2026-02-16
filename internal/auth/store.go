package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kraken-auth/internal/logger"

	"github.com/lib/pq"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	PasswordAlgo string
}

type Store struct {
	db *sql.DB
}

type UserProfile struct {
	UserID      string
	FullName    sql.NullString
	Email       sql.NullString
	PhoneNumber sql.NullString
	Address     sql.NullString
	CountryCode sql.NullString
	Roles       []string
}

type Role struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Country struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Restricted bool   `json:"restricted"`
	CanShip    bool   `json:"can_ship"`
}

type UserRow struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Email       string   `json:"email"`
	Country     string   `json:"country"`
	Roles       []string `json:"roles"`
	PhoneNumber string   `json:"phone_number"`
	Address     string   `json:"address"`
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	logger.Debugf("[store] GetUserByUsername start username=%q", username)

	const query = `
		SELECT u.id, u.username, ac.password_hash, ac.password_algo
		FROM auth.auth_credentials ac
		JOIN auth.users u ON ac.user_id = u.id
		WHERE u.username = $1
	`
	row := s.db.QueryRowContext(ctx, query, username)

	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.PasswordAlgo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Infof("[store] user not found username=%q", username)
			return nil, nil
		}
		wrapped := fmt.Errorf("GetUserByUsername query scan: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}

	logger.Debugf("[store] GetUserByUsername success username=%q id=%s algo=%s", username, u.ID, u.PasswordAlgo)
	return &u, nil
}

func (s *Store) GetUserProfileByUserID(ctx context.Context, userID string) (*UserProfile, error) {
	logger.Debugf("[store] GetUserProfileByUserID start user_id=%s", userID)

	var p UserProfile
	var roles pq.StringArray
	var phone sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, full_name, email, phone_number, address, country_code, roles
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&p.UserID, &p.FullName, &p.Email, &phone, &p.Address, &p.CountryCode, &roles,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Infof("[store] no profile found user_id=%s", userID)
			return nil, nil
		}
		wrapped := fmt.Errorf("GetUserProfileByUserID scan user_id=%s: %w", userID, err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}

	p.PhoneNumber = phone
	p.Roles = roles

	logger.Debugf("[store] GetUserProfileByUserID success user_id=%s roles=%d", userID, len(p.Roles))
	return &p, nil
}

func (s *Store) ListRoles(ctx context.Context) ([]Role, error) {
	logger.Debugf("[store] ListRoles start")

	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM auth.roles ORDER BY id`)
	if err != nil {
		wrapped := fmt.Errorf("ListRoles query: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}
	defer rows.Close()

	out := []Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			wrapped := fmt.Errorf("ListRoles scan: %w", err)
			logger.Errorf("[store] %v", wrapped)
			return nil, wrapped
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		wrapped := fmt.Errorf("ListRoles rows.Err: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}

	logger.Debugf("[store] ListRoles success count=%d", len(out))
	return out, nil
}

func (s *Store) ListCountries(ctx context.Context) ([]Country, error) {
	logger.Debugf("[store] ListCountries start")

	rows, err := s.db.QueryContext(ctx, `SELECT code, name, restricted, can_ship FROM public.countries ORDER BY name`)
	if err != nil {
		wrapped := fmt.Errorf("ListCountries query: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}
	defer rows.Close()

	out := []Country{}
	for rows.Next() {
		var c Country
		if err := rows.Scan(&c.Code, &c.Name, &c.Restricted, &c.CanShip); err != nil {
			wrapped := fmt.Errorf("ListCountries scan: %w", err)
			logger.Errorf("[store] %v", wrapped)
			return nil, wrapped
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		wrapped := fmt.Errorf("ListCountries rows.Err: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}

	logger.Debugf("[store] ListCountries success count=%d", len(out))
	return out, nil
}

func (s *Store) ListUsersWithProfile(ctx context.Context, limit, offset int) ([]UserRow, error) {
	logger.Debugf("[store] ListUsersWithProfile start limit=%d offset=%d", limit, offset)

	rows, err := s.db.QueryContext(ctx, `
SELECT u.id, u.username,
       COALESCE(p.full_name, '')        AS full_name,
       COALESCE(p.email, '')            AS email,
       COALESCE(p.phone_number, '')     AS phone_number,
       COALESCE(p.address, '')          AS address,
       COALESCE(p.country_code, '')     AS country,
       COALESCE(p.roles, '{}'::text[])  AS roles
FROM auth.users u
LEFT JOIN public.user_profiles p ON p.user_id = u.id
ORDER BY u.username
LIMIT $1 OFFSET $2
`, limit, offset)
	if err != nil {
		wrapped := fmt.Errorf("ListUsersWithProfile query: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}
	defer rows.Close()

	out := []UserRow{}
	for rows.Next() {
		var ur UserRow
		var roles pq.StringArray
		if err := rows.Scan(
			&ur.UserID, &ur.Username,
			&ur.FullName, &ur.Email,
			&ur.PhoneNumber, &ur.Address,
			&ur.Country, &roles,
		); err != nil {
			wrapped := fmt.Errorf("ListUsersWithProfile scan: %w", err)
			logger.Errorf("[store] %v", wrapped)
			return nil, wrapped
		}
		ur.Roles = []string(roles)
		out = append(out, ur)
	}
	if err := rows.Err(); err != nil {
		wrapped := fmt.Errorf("ListUsersWithProfile rows.Err: %w", err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}

	logger.Debugf("[store] ListUsersWithProfile success count=%d", len(out))
	return out, nil
}

// GetUserWithProfile returns one row for the right-side detail sheet.
func (s *Store) GetUserWithProfile(ctx context.Context, userID string) (*UserRow, error) {
	logger.Debugf("[store] GetUserWithProfile start user_id=%s", userID)

	row := s.db.QueryRowContext(ctx, `
SELECT u.id, u.username,
       COALESCE(p.full_name, '')        AS full_name,
       COALESCE(p.email, '')            AS email,
       COALESCE(p.phone_number, '')     AS phone_number,
       COALESCE(p.address, '')          AS address,
       COALESCE(p.country_code, '')     AS country,
       COALESCE(p.roles, '{}'::text[])  AS roles
FROM auth.users u
LEFT JOIN public.user_profiles p ON p.user_id = u.id
WHERE u.id = $1
`, userID)

	var ur UserRow
	var roles pq.StringArray
	if err := row.Scan(
		&ur.UserID, &ur.Username,
		&ur.FullName, &ur.Email,
		&ur.PhoneNumber, &ur.Address,
		&ur.Country, &roles,
	); err != nil {
		wrapped := fmt.Errorf("GetUserWithProfile scan user_id=%s: %w", userID, err)
		logger.Errorf("[store] %v", wrapped)
		return nil, wrapped
	}
	ur.Roles = []string(roles)

	logger.Debugf("[store] GetUserWithProfile success user_id=%s roles=%d", userID, len(ur.Roles))
	return &ur, nil
}

// CreateUser creates a new user with auth credentials and profile
func (s *Store) CreateUser(ctx context.Context, username, passwordHash, passwordAlgo string, profile *UserProfile) (string, error) {
	logger.Debugf("[store] CreateUser start username=%q", username)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("CreateUser begin tx: %w", err)
	}
	defer tx.Rollback()

	// Create user in auth.users
	var userID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO auth.users (id, username, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, NOW(), NOW())
		RETURNING id
	`, username).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("CreateUser insert users: %w", err)
	}

	// Create auth credentials
	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth.auth_credentials (id, user_id, password_hash, password_algo, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW(), NOW())
	`, userID, passwordHash, passwordAlgo)
	if err != nil {
		return "", fmt.Errorf("CreateUser insert credentials: %w", err)
	}

	// Create profile if provided
	if profile != nil {
		roles := pq.StringArray(profile.Roles)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO public.user_profiles (user_id, full_name, email, phone_number, address, country_code, roles)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, profile.FullName, profile.Email, profile.PhoneNumber, profile.Address, profile.CountryCode, roles)
		if err != nil {
			return "", fmt.Errorf("CreateUser insert profile: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("CreateUser commit: %w", err)
	}

	logger.Infof("[store] CreateUser success username=%q user_id=%s", username, userID)
	return userID, nil
}

// UpdateUserProfile updates user profile information
func (s *Store) UpdateUserProfile(ctx context.Context, userID string, fullName, email, phoneNumber, address, countryCode sql.NullString) error {
	logger.Debugf("[store] UpdateUserProfile start user_id=%s", userID)

	_, err := s.db.ExecContext(ctx, `
		UPDATE public.user_profiles
		SET full_name = $2, email = $3, phone_number = $4, address = $5, country_code = $6
		WHERE user_id = $1
	`, userID, fullName, email, phoneNumber, address, countryCode)
	if err != nil {
		return fmt.Errorf("UpdateUserProfile: %w", err)
	}

	logger.Infof("[store] UpdateUserProfile success user_id=%s", userID)
	return nil
}

// UpdateUserRoles updates user roles
func (s *Store) UpdateUserRoles(ctx context.Context, userID string, roles []string) error {
	logger.Debugf("[store] UpdateUserRoles start user_id=%s roles=%v", userID, roles)

	_, err := s.db.ExecContext(ctx, `
		UPDATE public.user_profiles
		SET roles = $2
		WHERE user_id = $1
	`, userID, pq.StringArray(roles))
	if err != nil {
		return fmt.Errorf("UpdateUserRoles: %w", err)
	}

	logger.Infof("[store] UpdateUserRoles success user_id=%s count=%d", userID, len(roles))
	return nil
}

// UpdateUserPassword updates user password
func (s *Store) UpdateUserPassword(ctx context.Context, userID, passwordHash, passwordAlgo string) error {
	logger.Debugf("[store] UpdateUserPassword start user_id=%s algo=%s", userID, passwordAlgo)

	_, err := s.db.ExecContext(ctx, `
		UPDATE auth.auth_credentials
		SET password_hash = $2, password_algo = $3, updated_at = NOW()
		WHERE user_id = $1
	`, userID, passwordHash, passwordAlgo)
	if err != nil {
		return fmt.Errorf("UpdateUserPassword: %w", err)
	}

	logger.Infof("[store] UpdateUserPassword success user_id=%s", userID)
	return nil
}
