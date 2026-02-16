package auth

import (
	"database/sql"
	"testing"
)

func TestNewStore(t *testing.T) {
	// Create a mock DB (we won't actually use it)
	var db *sql.DB

	store := NewStore(db)
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.db != db {
		t.Error("NewStore() did not set db field correctly")
	}
}

func TestUserStruct(t *testing.T) {
	user := User{
		ID:           "123",
		Username:     "testuser",
		PasswordHash: "hash123",
		PasswordAlgo: "bcrypt",
	}

	if user.ID != "123" {
		t.Errorf("User.ID = %q, want %q", user.ID, "123")
	}
	if user.Username != "testuser" {
		t.Errorf("User.Username = %q, want %q", user.Username, "testuser")
	}
	if user.PasswordHash != "hash123" {
		t.Errorf("User.PasswordHash = %q, want %q", user.PasswordHash, "hash123")
	}
	if user.PasswordAlgo != "bcrypt" {
		t.Errorf("User.PasswordAlgo = %q, want %q", user.PasswordAlgo, "bcrypt")
	}
}

func TestUserProfileStruct(t *testing.T) {
	profile := UserProfile{
		UserID:      "user123",
		FullName:    sql.NullString{String: "Test User", Valid: true},
		Email:       sql.NullString{String: "test@example.com", Valid: true},
		PhoneNumber: sql.NullString{String: "+1234567890", Valid: true},
		Address:     sql.NullString{String: "123 Main St", Valid: true},
		CountryCode: sql.NullString{String: "US", Valid: true},
		Roles:       []string{"admin", "user"},
	}

	if profile.UserID != "user123" {
		t.Errorf("UserProfile.UserID = %q, want %q", profile.UserID, "user123")
	}
	if !profile.FullName.Valid || profile.FullName.String != "Test User" {
		t.Errorf("UserProfile.FullName = %v, want Test User", profile.FullName)
	}
	if len(profile.Roles) != 2 {
		t.Errorf("UserProfile.Roles length = %d, want 2", len(profile.Roles))
	}
}

func TestUserProfileNullFields(t *testing.T) {
	profile := UserProfile{
		UserID:      "user456",
		FullName:    sql.NullString{Valid: false},
		Email:       sql.NullString{Valid: false},
		PhoneNumber: sql.NullString{Valid: false},
		Address:     sql.NullString{Valid: false},
		CountryCode: sql.NullString{Valid: false},
		Roles:       []string{},
	}

	if profile.FullName.Valid {
		t.Error("UserProfile.FullName should be invalid (NULL)")
	}
	if profile.Email.Valid {
		t.Error("UserProfile.Email should be invalid (NULL)")
	}
	if profile.PhoneNumber.Valid {
		t.Error("UserProfile.PhoneNumber should be invalid (NULL)")
	}
}

func TestRoleStruct(t *testing.T) {
	role := Role{
		ID:   1,
		Name: "admin",
	}

	if role.ID != 1 {
		t.Errorf("Role.ID = %d, want 1", role.ID)
	}
	if role.Name != "admin" {
		t.Errorf("Role.Name = %q, want %q", role.Name, "admin")
	}
}

func TestUserRowStruct(t *testing.T) {
	userRow := UserRow{
		UserID:      "user789",
		Username:    "testuser",
		FullName:    "Test User",
		Email:       "test@example.com",
		Country:     "US",
		Roles:       []string{"admin", "viewer"},
		PhoneNumber: "+1234567890",
		Address:     "123 Main St",
	}

	if userRow.UserID != "user789" {
		t.Errorf("UserRow.UserID = %q, want %q", userRow.UserID, "user789")
	}
	if userRow.Username != "testuser" {
		t.Errorf("UserRow.Username = %q, want %q", userRow.Username, "testuser")
	}
	if len(userRow.Roles) != 2 {
		t.Errorf("UserRow.Roles length = %d, want 2", len(userRow.Roles))
	}
	if userRow.Email != "test@example.com" {
		t.Errorf("UserRow.Email = %q, want %q", userRow.Email, "test@example.com")
	}
}

func TestUserRowWithEmptyFields(t *testing.T) {
	userRow := UserRow{
		UserID:      "user999",
		Username:    "minimaluser",
		FullName:    "",
		Email:       "",
		Country:     "",
		Roles:       []string{},
		PhoneNumber: "",
		Address:     "",
	}

	if userRow.UserID != "user999" {
		t.Errorf("UserRow.UserID = %q, want %q", userRow.UserID, "user999")
	}
	if userRow.FullName != "" {
		t.Errorf("UserRow.FullName should be empty")
	}
	if len(userRow.Roles) != 0 {
		t.Errorf("UserRow.Roles should be empty")
	}
}

func TestStoreStructure(t *testing.T) {
	t.Run("store has db field", func(t *testing.T) {
		var db *sql.DB
		store := &Store{db: db}

		if store.db != db {
			t.Error("Store.db field not set correctly")
		}
	})

	t.Run("store can be created with nil db", func(t *testing.T) {
		store := NewStore(nil)
		if store == nil {
			t.Fatal("NewStore(nil) should not return nil store")
		}
		if store.db != nil {
			t.Error("NewStore(nil) should create store with nil db")
		}
	})
}

func TestMultipleRoles(t *testing.T) {
	roles := []string{"admin", "user", "viewer", "editor", "owner"}
	userRow := UserRow{
		UserID:   "multirole",
		Username: "multiroleuser",
		Roles:    roles,
	}

	if len(userRow.Roles) != 5 {
		t.Errorf("UserRow.Roles length = %d, want 5", len(userRow.Roles))
	}

	for i, role := range roles {
		if userRow.Roles[i] != role {
			t.Errorf("UserRow.Roles[%d] = %q, want %q", i, userRow.Roles[i], role)
		}
	}
}
