package middleware

import (
	"fmt"
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
)

// roleHierarchy defines the permission level for each role.
// Higher number = more permissions.
var roleHierarchy = map[string]int{
	"member": 1,
	"admin":  2,
	"owner":  3,
}

// RequireRole returns middleware that checks the user's role meets the minimum required.
// The role check uses a hierarchy: owner > admin > member.
// Pass the minimum required role(s) — user passes if their role >= any of the required roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	// Find minimum required level from the provided roles
	minLevel := 999
	for _, role := range roles {
		if level, ok := roleHierarchy[role]; ok && level < minLevel {
			minLevel = level
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := auth.GetRole(r.Context())
			if !ok {
				http.Error(w, `{"error":"missing role context"}`, http.StatusUnauthorized)
				return
			}

			userLevel, ok := roleHierarchy[userRole]
			if !ok {
				http.Error(w, `{"error":"unknown role"}`, http.StatusForbidden)
				return
			}

			if userLevel < minLevel {
				msg := fmt.Sprintf(`{"error":"insufficient permissions: requires %s role or higher"}`, roles[0])
				http.Error(w, msg, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
