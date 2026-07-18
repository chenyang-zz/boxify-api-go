package realdb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	dbpostgres "github.com/boxify/api-go/internal/infrastructure/db/postgres"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	realDBInitialPassword = "Cove-realdb-123!"
	realDBNewPassword     = "Cove-realdb-456!"
)

type userProfileData struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Nickname *string   `json:"nickname"`
	Email    *string   `json:"email"`
}

// TestProfileAndPasswordPersistenceAndUserIsolation 验证真实 API 与 PostgreSQL 下的资料持久化、密码拒绝与轮换，以及另一用户数据不受影响。
func TestProfileAndPasswordPersistenceAndUserIsolation(t *testing.T) {
	apiURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_API_URL"), "/")
	databaseURL := os.Getenv("COVE_REAL_DB_DATABASE_URL")
	if apiURL == "" || databaseURL == "" {
		t.Skip("COVE_REAL_DB_API_URL and COVE_REAL_DB_DATABASE_URL are required")
	}

	ctx := t.Context()
	db, err := dbpostgres.NewGormDB(ctx, dbpostgres.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("NewGormDB error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("Close DB error = %v", closeErr)
		}
	})

	client := &http.Client{Timeout: 10 * time.Second}
	runID := os.Getenv("COVE_REAL_DB_RUN_ID")
	owner := registerUser(t, client, apiURL, testUsername("profile-owner", runID))
	other := registerUser(t, client, apiURL, testUsername("profile-other", runID))
	t.Cleanup(func() {
		result := db.WithContext(context.Background()).Where("id IN ?", []uuid.UUID{owner.UserID, other.UserID}).Delete(&models.User{})
		if result.Error != nil {
			t.Errorf("cleanup profile users error = %v", result.Error)
		}
	})

	var initialOwner models.User
	if err := db.WithContext(ctx).Where("id = ?", owner.UserID).First(&initialOwner).Error; err != nil {
		t.Fatalf("query initial owner error = %v", err)
	}
	var initialOther models.User
	if err := db.WithContext(ctx).Where("id = ?", other.UserID).First(&initialOther).Error; err != nil {
		t.Fatalf("query initial other user error = %v", err)
	}

	nickname := "Local Database Owner"
	updatedEmail := fmt.Sprintf("profile-%s@example.test", uuid.NewString())
	profile := doJSON[userProfileData](
		t,
		client,
		http.MethodPut,
		apiURL+"/api/auth/profile",
		owner.AccessToken,
		map[string]string{
			"nickname": "  " + nickname + "  ",
			"email":    strings.ToUpper(updatedEmail),
		},
		http.StatusOK,
	)
	if profile.Code != 0 {
		t.Fatalf("profile update code = %d, want 0", profile.Code)
	}
	assertUserProfile(t, &profile.Data, owner.UserID, owner.Username, &nickname, &updatedEmail)
	assertCurrentUser(t, client, apiURL, owner.AccessToken, owner.UserID, owner.Username, &nickname, &updatedEmail)
	assertCurrentUser(t, client, apiURL, other.AccessToken, other.UserID, other.Username, nil, initialOther.Email)

	var persistedOwner models.User
	if err := db.WithContext(ctx).Where("id = ?", owner.UserID).First(&persistedOwner).Error; err != nil {
		t.Fatalf("query persisted owner profile error = %v", err)
	}
	if persistedOwner.Nickname == nil || *persistedOwner.Nickname != nickname || persistedOwner.Email == nil || *persistedOwner.Email != updatedEmail {
		t.Fatalf("persisted owner profile nickname=%v email=%v, want %q and %q", persistedOwner.Nickname, persistedOwner.Email, nickname, updatedEmail)
	}
	if persistedOwner.PasswordHash != initialOwner.PasswordHash {
		t.Fatal("profile update unexpectedly changed the password hash")
	}

	wrongCurrent := doJSON[json.RawMessage](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/password",
		owner.AccessToken,
		map[string]string{"old_password": "incorrect-current-password", "new_password": realDBNewPassword},
		http.StatusBadRequest,
	)
	if wrongCurrent.Code != 40000 || wrongCurrent.Message != "原密码错误" {
		t.Fatalf("incorrect-current-password response code=%d message=%q, want 40000 原密码错误", wrongCurrent.Code, wrongCurrent.Message)
	}
	assertPersistedPasswordHash(t, db, owner.UserID, initialOwner.PasswordHash, realDBInitialPassword, false)

	reusedPassword := doJSON[json.RawMessage](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/password",
		owner.AccessToken,
		map[string]string{"old_password": realDBInitialPassword, "new_password": realDBInitialPassword},
		http.StatusBadRequest,
	)
	if reusedPassword.Code != 40000 || reusedPassword.Message != "原密码与新密码不能相同" {
		t.Fatalf("reused-password response code=%d message=%q, want 40000 原密码与新密码不能相同", reusedPassword.Code, reusedPassword.Message)
	}
	assertPersistedPasswordHash(t, db, owner.UserID, initialOwner.PasswordHash, realDBInitialPassword, false)

	changed := doJSON[json.RawMessage](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/password",
		owner.AccessToken,
		map[string]string{"old_password": realDBInitialPassword, "new_password": realDBNewPassword},
		http.StatusOK,
	)
	if changed.Code != 0 {
		t.Fatalf("password change code = %d, want 0", changed.Code)
	}
	assertPersistedPasswordHash(t, db, owner.UserID, initialOwner.PasswordHash, realDBNewPassword, true)

	oldLogin := doJSON[json.RawMessage](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/login",
		"",
		map[string]string{"login": owner.Username, "password": realDBInitialPassword},
		http.StatusUnauthorized,
	)
	if oldLogin.Code != 40103 {
		t.Fatalf("old-password login code = %d, want 40103", oldLogin.Code)
	}

	newLogin := loginUser(t, client, apiURL, owner.Username, realDBNewPassword)
	if newLogin.UserID != owner.UserID || newLogin.AccessToken == "" {
		t.Fatalf("new-password login user_id=%s access_token_present=%v, want owner %s", newLogin.UserID, newLogin.AccessToken != "", owner.UserID)
	}
	assertCurrentUser(t, client, apiURL, newLogin.AccessToken, owner.UserID, owner.Username, &nickname, &updatedEmail)

	otherLogin := loginUser(t, client, apiURL, other.Username, realDBInitialPassword)
	if otherLogin.UserID != other.UserID || otherLogin.AccessToken == "" {
		t.Fatalf("other-user login user_id=%s access_token_present=%v, want other user %s", otherLogin.UserID, otherLogin.AccessToken != "", other.UserID)
	}
	var persistedOther models.User
	if err := db.WithContext(ctx).Where("id = ?", other.UserID).First(&persistedOther).Error; err != nil {
		t.Fatalf("query persisted other user error = %v", err)
	}
	if persistedOther.Nickname != nil || persistedOther.Email == nil || initialOther.Email == nil || *persistedOther.Email != *initialOther.Email {
		t.Fatalf("other user profile changed: nickname=%v email=%v, want nickname nil email %v", persistedOther.Nickname, persistedOther.Email, initialOther.Email)
	}
	if persistedOther.PasswordHash != initialOther.PasswordHash || !security.CheckPassword(persistedOther.PasswordHash, realDBInitialPassword) {
		t.Fatal("other user's password hash changed or no longer matches the initial password")
	}
}

func assertCurrentUser(
	t *testing.T,
	client *http.Client,
	apiURL string,
	accessToken string,
	userID uuid.UUID,
	username string,
	nickname *string,
	email *string,
) {
	t.Helper()
	response := doJSON[userProfileData](t, client, http.MethodGet, apiURL+"/api/auth/me", accessToken, nil, http.StatusOK)
	if response.Code != 0 {
		t.Fatalf("current-user response code = %d, want 0", response.Code)
	}
	assertUserProfile(t, &response.Data, userID, username, nickname, email)
}

func assertUserProfile(t *testing.T, actual *userProfileData, userID uuid.UUID, username string, nickname *string, email *string) {
	t.Helper()
	if actual.ID != userID || actual.Username != username || !equalOptionalString(actual.Nickname, nickname) || !equalOptionalString(actual.Email, email) {
		t.Fatalf(
			"user profile id=%s username=%q nickname=%v email=%v, want id=%s username=%q nickname=%v email=%v",
			actual.ID,
			actual.Username,
			actual.Nickname,
			actual.Email,
			userID,
			username,
			nickname,
			email,
		)
	}
}

func equalOptionalString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func assertPersistedPasswordHash(t *testing.T, db *gorm.DB, userID uuid.UUID, initialHash string, expectedPassword string, wantChanged bool) {
	t.Helper()
	var user models.User
	if err := db.WithContext(t.Context()).Where("id = ?", userID).First(&user).Error; err != nil {
		t.Fatalf("query persisted password hash error = %v", err)
	}
	if (user.PasswordHash != initialHash) != wantChanged {
		t.Fatalf("password hash changed=%v, want %v", user.PasswordHash != initialHash, wantChanged)
	}
	if user.PasswordHash == expectedPassword || !security.CheckPassword(user.PasswordHash, expectedPassword) {
		t.Fatal("persisted password is plaintext or does not match the expected password")
	}
	if wantChanged && security.CheckPassword(user.PasswordHash, realDBInitialPassword) {
		t.Fatal("changed password hash still matches the initial password")
	}
}

func loginUser(t *testing.T, client *http.Client, apiURL string, login string, password string) *authData {
	t.Helper()
	response := doJSON[authData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/login",
		"",
		map[string]string{"login": login, "password": password},
		http.StatusOK,
	)
	if response.Code != 0 || response.Data.UserID == uuid.Nil || response.Data.AccessToken == "" {
		t.Fatalf(
			"login response code=%d user_id=%s access_token_present=%v, want authenticated user",
			response.Code,
			response.Data.UserID,
			response.Data.AccessToken != "",
		)
	}
	return &response.Data
}
