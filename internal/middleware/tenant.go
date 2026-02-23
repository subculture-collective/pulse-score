package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
)

// TenantIsolation returns middleware that ensures org_id is set in context.
// By default it uses org_id from JWT claims.
// If X-Organization-ID header is set, it verifies membership and switches.
func TenantIsolation(orgRepo *repository.OrganizationRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			userID, ok := auth.GetUserID(ctx)
			if !ok {
				http.Error(w, `{"error":"missing user context"}`, http.StatusUnauthorized)
				return
			}

			orgID, ok := auth.GetOrgID(ctx)
			if !ok {
				http.Error(w, `{"error":"missing organization context"}`, http.StatusUnauthorized)
				return
			}

			// Check for org switch via header
			if headerOrgID := r.Header.Get("X-Organization-ID"); headerOrgID != "" {
				switchOrgID, err := uuid.Parse(headerOrgID)
				if err != nil {
					http.Error(w, `{"error":"invalid X-Organization-ID header"}`, http.StatusBadRequest)
					return
				}

				if switchOrgID != orgID {
					// Verify user is a member of the target org
					isMember, err := orgRepo.IsMember(ctx, userID, switchOrgID)
					if err != nil {
						http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
						return
					}
					if !isMember {
						http.Error(w, `{"error":"you are not a member of this organization"}`, http.StatusForbidden)
						return
					}

					// Switch org context
					ctx = auth.WithOrgID(ctx, switchOrgID)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
