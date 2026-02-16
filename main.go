package main

// Fix: Prevent nil pointer panic when user not found
import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"kraken-auth/internal/auth"
	"kraken-auth/internal/db"
	"kraken-auth/internal/featureflags"
	mw "kraken-auth/internal/http/middleware"
	"kraken-auth/internal/logger"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func allowAdminOrStoreAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := auth.GetBearerToken(r)
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		if !auth.HasAnyRole(claims.Roles, "admin", "storeadmin") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unwrap(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func main() {
	// 1) DB init
	sqlDB, err := db.Init()
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer sqlDB.Close()

	// 2) Feature flags init (non-fatal)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := featureflags.Init(ctx, ""); err != nil {
		log.Printf("feature flags init warning: %v", err)
	} else {
		log.Printf("feature flags ready: offline=%v, logLevel=%s",
			featureflags.Values().Offline.IsEnabled(nil),
			featureflags.Values().LogLevel.GetValue(nil))
	}
	defer featureflags.Shutdown()

	// 2a) Initialize levelled logger from flag & watch for flips
	logger.Init(featureflags.Values().LogLevel.GetValue(nil))
	logger.Infof("log level set to %s", logger.GetLevel())

	go func() {
		prev := featureflags.Values().LogLevel.GetValue(nil)
		for {
			time.Sleep(5 * time.Second)
			cur := featureflags.Values().LogLevel.GetValue(nil)
			if cur != prev {
				logger.SetLevel(cur)
				logger.Infof("log level changed to %s", logger.GetLevel())
				prev = cur
			}
		}
	}()

	// 3) Required secrets
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	// 4) Router
	r := mux.NewRouter()

	// 4a) Offline kill-switch middleware (placed immediately after router creation)
	offlineGate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// always allow health checks
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}
			// block all other requests when Offline flag is ON
			if featureflags.Values().Offline.IsEnabled(nil) {
				http.Error(w, "service temporarily offline", http.StatusServiceUnavailable)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	r.Use(offlineGate)

	// 4b) Request logger (skip noisy health endpoints)
	r.Use(mw.LogRequests(mw.WithSkips("/health", "/ready")))

	// 5) Health endpoints
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet)

	r.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if err := sqlDB.Ping(); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}).Methods(http.MethodGet)

	// 6) Inspect current flag values
	r.HandleFunc("/_flags", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"offline":  featureflags.Values().Offline.IsEnabled(nil),
			"logLevel": featureflags.Values().LogLevel.GetValue(nil),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}).Methods(http.MethodGet)

	// 7) Auth endpoints
	store := auth.NewStore(sqlDB)

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var req loginReq
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&req); err != nil {
			ct := r.Header.Get("Content-Type")
			cl := r.Header.Get("Content-Length")
			logger.Warnf("[login] bad json: decode=%v, Content-Type=%q Content-Length=%q", err, ct, cl)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUsername(r.Context(), req.Username)
		if err != nil || user == nil {
			logger.Warnf("[login] invalid credentials for user=%q", req.Username)
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		if !auth.CheckPassword(req.Password, user.PasswordAlgo, user.PasswordHash) {
			logger.Warnf("[login] invalid password for user=%q", req.Username)
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		profile, err := store.GetUserProfileByUserID(r.Context(), user.ID)
		if err != nil || profile == nil {
			logger.Warnf("[login] no profile found for user_id=%s", user.ID)
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		// Extract country from profile
		country := ""
		if profile.CountryCode.Valid {
			country = profile.CountryCode.String
		}

		token, err := auth.GenerateToken(*user, profile.Roles, country, jwtSecret)
		if err != nil {
			logger.Errorf("[login] token generation failed: %v", err)
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"id":           profile.UserID,
			"user_id":      user.ID,
			"username":     user.Username,
			"full_name":    unwrap(profile.FullName),
			"email":        unwrap(profile.Email),
			"phone_number": unwrap(profile.PhoneNumber),
			"address":      unwrap(profile.Address),
			"country":      unwrap(profile.CountryCode),
			"roles":        profile.Roles,
			"token":        token,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}).Methods(http.MethodPost)

	// 8) Admin API (admin OR storeadmin)
	r.Handle("/admin/roles", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles, err := store.ListRoles(r.Context())
		if err != nil {
			logger.Errorf("[admin/roles] list roles failed: %v", err)
			http.Error(w, "list roles failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(roles)
	}))).Methods(http.MethodGet)

	r.Handle("/admin/countries", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countries, err := store.ListCountries(r.Context())
		if err != nil {
			logger.Errorf("[admin/countries] list countries failed: %v", err)
			http.Error(w, "list countries failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(countries)
	}))).Methods(http.MethodGet)

	r.Handle("/admin/users", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := 200
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		users, err := store.ListUsersWithProfile(r.Context(), limit, offset)
		if err != nil {
			logger.Errorf("[admin/users] list users failed: %v", err)
			http.Error(w, "list users failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(users)
	}))).Methods(http.MethodGet)

	r.Handle("/admin/users/{id}", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		u, err := store.GetUserWithProfile(r.Context(), id)
		if err != nil {
			logger.Errorf("[admin/users/%s] get user failed: %v", id, err)
			http.Error(w, "get user failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(u)
	}))).Methods(http.MethodGet)

	// Create new user
	r.Handle("/admin/users", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username    string   `json:"username"`
			Password    string   `json:"password"`
			FullName    string   `json:"full_name"`
			Email       string   `json:"email"`
			PhoneNumber string   `json:"phone_number"`
			Address     string   `json:"address"`
			Country     string   `json:"country"`
			Roles       []string `json:"roles"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "username and password required", http.StatusBadRequest)
			return
		}

		// Hash password
		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			logger.Errorf("[admin/create-user] hash password failed: %v", err)
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		// Create profile
		profile := &auth.UserProfile{
			FullName:    toNullString(req.FullName),
			Email:       toNullString(req.Email),
			PhoneNumber: toNullString(req.PhoneNumber),
			Address:     toNullString(req.Address),
			CountryCode: toNullString(req.Country),
			Roles:       req.Roles,
		}

		userID, err := store.CreateUser(r.Context(), req.Username, passwordHash, "bcrypt", profile)
		if err != nil {
			logger.Errorf("[admin/create-user] create failed: %v", err)
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"user_id": userID})
	}))).Methods(http.MethodPost)

	// Update user profile
	r.Handle("/admin/users/{id}", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var req struct {
			FullName    string `json:"full_name"`
			Email       string `json:"email"`
			PhoneNumber string `json:"phone_number"`
			Address     string `json:"address"`
			Country     string `json:"country"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		err := store.UpdateUserProfile(r.Context(), id,
			toNullString(req.FullName),
			toNullString(req.Email),
			toNullString(req.PhoneNumber),
			toNullString(req.Address),
			toNullString(req.Country),
		)
		if err != nil {
			logger.Errorf("[admin/update-user/%s] update failed: %v", id, err)
			http.Error(w, "failed to update user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))).Methods(http.MethodPut)

	// Update user roles
	r.Handle("/admin/users/{id}/roles", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var req struct {
			Roles []string `json:"roles"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		err := store.UpdateUserRoles(r.Context(), id, req.Roles)
		if err != nil {
			logger.Errorf("[admin/update-roles/%s] update failed: %v", id, err)
			http.Error(w, "failed to update roles", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))).Methods(http.MethodPut)

	// Reset user password
	r.Handle("/admin/users/{id}/password", allowAdminOrStoreAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var req struct {
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Password == "" {
			http.Error(w, "password required", http.StatusBadRequest)
			return
		}

		// Hash password
		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			logger.Errorf("[admin/reset-password/%s] hash failed: %v", id, err)
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		err = store.UpdateUserPassword(r.Context(), id, passwordHash, "bcrypt")
		if err != nil {
			logger.Errorf("[admin/reset-password/%s] update failed: %v", id, err)
			http.Error(w, "failed to update password", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))).Methods(http.MethodPut)

	// 9) HTTP server
	s := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Infof("kraken-auth listening on %s", s.Addr)
	log.Fatal(s.ListenAndServe())
}
