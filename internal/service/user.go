package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// UserProfileResponse is returned by user profile endpoints.
type UserProfileResponse struct {
	ID        uuid.UUID          `json:"id"`
	Email     string             `json:"email"`
	FirstName string             `json:"first_name"`
	LastName  string             `json:"last_name"`
	AvatarURL string             `json:"avatar_url"`
	Orgs      []UserOrgMembership `json:"organizations"`
}

// UserOrgMembership represents an org the user belongs to.
type UserOrgMembership struct {
	OrgID uuid.UUID `json:"org_id"`
	Name  string    `json:"name"`
	Slug  string    `json:"slug"`
	Role  string    `json:"role"`
}

// UpdateProfileRequest holds fields to update on a user profile.
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	AvatarURL *string `json:"avatar_url"`
}

// UserService handles user profile logic.
type UserService struct {
	users *repository.UserRepository
	orgs  *repository.OrganizationRepository
}

// NewUserService creates a new UserService.
func NewUserService(users *repository.UserRepository, orgs *repository.OrganizationRepository) *UserService {
	return &UserService{users: users, orgs: orgs}
}

// GetProfile returns the authenticated user's profile with org memberships.
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, &NotFoundError{Resource: "user", Message: "user not found"}
	}

	userOrgs, err := s.users.GetUserOrgs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orgs: %w", err)
	}

	var memberships []UserOrgMembership
	for _, uo := range userOrgs {
		org, err := s.orgs.GetByID(ctx, uo.OrgID)
		if err != nil {
			return nil, fmt.Errorf("get org: %w", err)
		}
		if org != nil {
			memberships = append(memberships, UserOrgMembership{
				OrgID: org.ID,
				Name:  org.Name,
				Slug:  org.Slug,
				Role:  uo.Role,
			})
		}
	}

	return &UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		AvatarURL: user.AvatarURL,
		Orgs:      memberships,
	}, nil
}

// UpdateProfile updates the user's profile fields.
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*UserProfileResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, &NotFoundError{Resource: "user", Message: "user not found"}
	}

	firstName := user.FirstName
	lastName := user.LastName
	avatarURL := user.AvatarURL

	if req.FirstName != nil {
		firstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		lastName = strings.TrimSpace(*req.LastName)
	}
	if req.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*req.AvatarURL)
	}

	if err := s.users.UpdateProfile(ctx, userID, firstName, lastName, avatarURL); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return s.GetProfile(ctx, userID)
}
