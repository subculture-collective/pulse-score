package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// MemberService handles organization member management.
type MemberService struct {
	orgRepo *repository.OrganizationRepository
}

// NewMemberService creates a new MemberService.
func NewMemberService(orgRepo *repository.OrganizationRepository) *MemberService {
	return &MemberService{orgRepo: orgRepo}
}

// MemberResponse represents a member in the response.
type MemberResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

// UpdateRoleRequest holds input for updating a member's role.
type UpdateRoleRequest struct {
	Role string `json:"role"`
}

// ListMembers returns all members of an organization.
func (s *MemberService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]MemberResponse, error) {
	members, err := s.orgRepo.ListMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	result := make([]MemberResponse, len(members))
	for i, m := range members {
		result[i] = MemberResponse{
			UserID:    m.UserID,
			Email:     m.Email,
			FirstName: m.FirstName,
			LastName:  m.LastName,
			AvatarURL: m.AvatarURL,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
		}
	}

	return result, nil
}

// UpdateRole updates a member's role.
func (s *MemberService) UpdateRole(ctx context.Context, orgID, callerID, targetUserID uuid.UUID, req UpdateRoleRequest) error {
	validRoles := map[string]bool{"owner": true, "admin": true, "member": true}
	if !validRoles[req.Role] {
		return &ValidationError{Field: "role", Message: "role must be owner, admin, or member"}
	}

	// Can't change own role
	if callerID == targetUserID {
		return &ForbiddenError{Message: "cannot change your own role"}
	}

	// Check target is a member
	currentRole, err := s.orgRepo.GetMemberRole(ctx, orgID, targetUserID)
	if err != nil {
		return fmt.Errorf("get member role: %w", err)
	}
	if currentRole == "" {
		return &NotFoundError{Resource: "member", Message: "user is not a member of this organization"}
	}

	// Demoting from owner — ensure at least one owner remains
	if currentRole == "owner" && req.Role != "owner" {
		ownerCount, err := s.orgRepo.CountOwners(ctx, orgID)
		if err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if ownerCount <= 1 {
			return &ValidationError{Field: "role", Message: "cannot demote the last owner"}
		}
	}

	if err := s.orgRepo.UpdateMemberRole(ctx, orgID, targetUserID, req.Role); err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	return nil
}

// RemoveMember removes a member from the organization.
func (s *MemberService) RemoveMember(ctx context.Context, orgID, callerID, targetUserID uuid.UUID) error {
	if callerID == targetUserID {
		return &ForbiddenError{Message: "cannot remove yourself"}
	}

	currentRole, err := s.orgRepo.GetMemberRole(ctx, orgID, targetUserID)
	if err != nil {
		return fmt.Errorf("get member role: %w", err)
	}
	if currentRole == "" {
		return &NotFoundError{Resource: "member", Message: "user is not a member of this organization"}
	}

	// Can't remove the last owner
	if currentRole == "owner" {
		ownerCount, err := s.orgRepo.CountOwners(ctx, orgID)
		if err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if ownerCount <= 1 {
			return &ValidationError{Field: "user_id", Message: "cannot remove the last owner"}
		}
	}

	if err := s.orgRepo.RemoveMember(ctx, orgID, targetUserID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}
