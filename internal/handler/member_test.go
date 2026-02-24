package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockMemberService struct {
	listMembersFn  func(ctx context.Context, orgID uuid.UUID) ([]service.MemberResponse, error)
	updateRoleFn   func(ctx context.Context, orgID, callerID, targetUserID uuid.UUID, req service.UpdateRoleRequest) error
	removeMemberFn func(ctx context.Context, orgID, callerID, targetUserID uuid.UUID) error
}

func (m *mockMemberService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]service.MemberResponse, error) {
	return m.listMembersFn(ctx, orgID)
}

func (m *mockMemberService) UpdateRole(ctx context.Context, orgID, callerID, targetUserID uuid.UUID, req service.UpdateRoleRequest) error {
	return m.updateRoleFn(ctx, orgID, callerID, targetUserID, req)
}

func (m *mockMemberService) RemoveMember(ctx context.Context, orgID, callerID, targetUserID uuid.UUID) error {
	return m.removeMemberFn(ctx, orgID, callerID, targetUserID)
}

func TestMemberList_Unauthorized(t *testing.T) {
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMemberList_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockMemberService{
		listMembersFn: func(ctx context.Context, oID uuid.UUID) ([]service.MemberResponse, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return []service.MemberResponse{
				{Email: "alice@example.com", Role: "owner"},
				{Email: "bob@example.com", Role: "member"},
			}, nil
		},
	}

	h := NewMemberHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string][]service.MemberResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp["members"]) != 2 {
		t.Errorf("expected 2 members, got %d", len(resp["members"]))
	}
}

func TestMemberUpdateRole_Unauthorized_NoOrgID(t *testing.T) {
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+uuid.New().String()+"/role", nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_Unauthorized_NoUserID(t *testing.T) {
	orgID := uuid.New()
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+uuid.New().String()+"/role", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/not-valid/role", nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", "not-valid")
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_InvalidBody(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+targetID.String()+"/role", bytes.NewBufferString("{invalid"))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", targetID.String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_Success(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()
	mock := &mockMemberService{
		updateRoleFn: func(ctx context.Context, oID, cID, tID uuid.UUID, req service.UpdateRoleRequest) error {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			if cID != callerID {
				t.Errorf("expected callerID %s, got %s", callerID, cID)
			}
			if tID != targetID {
				t.Errorf("expected targetID %s, got %s", targetID, tID)
			}
			if req.Role != "admin" {
				t.Errorf("expected role admin, got %s", req.Role)
			}
			return nil
		},
	}

	h := NewMemberHandler(mock)
	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+targetID.String()+"/role", bytes.NewReader(body))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, callerID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", targetID.String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_ValidationError(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()
	mock := &mockMemberService{
		updateRoleFn: func(ctx context.Context, oID, cID, tID uuid.UUID, req service.UpdateRoleRequest) error {
			return &service.ValidationError{Field: "role", Message: "role must be owner, admin, or member"}
		},
	}

	h := NewMemberHandler(mock)
	body, _ := json.Marshal(map[string]string{"role": "superadmin"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+targetID.String()+"/role", bytes.NewReader(body))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, callerID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", targetID.String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestMemberUpdateRole_ForbiddenSelf(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()
	mock := &mockMemberService{
		updateRoleFn: func(ctx context.Context, oID, cID, tID uuid.UUID, req service.UpdateRoleRequest) error {
			return &service.ForbiddenError{Message: "cannot change your own role"}
		},
	}

	h := NewMemberHandler(mock)
	body, _ := json.Marshal(map[string]string{"role": "member"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/members/"+callerID.String()+"/role", bytes.NewReader(body))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, callerID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", callerID.String())
	rr := httptest.NewRecorder()

	h.UpdateRole(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestMemberRemove_Unauthorized(t *testing.T) {
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/members/"+uuid.New().String(), nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.Remove(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMemberRemove_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	h := NewMemberHandler(&mockMemberService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/members/bad", nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", "bad")
	rr := httptest.NewRecorder()

	h.Remove(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestMemberRemove_Success(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()
	mock := &mockMemberService{
		removeMemberFn: func(ctx context.Context, oID, cID, tID uuid.UUID) error {
			if oID != orgID || cID != callerID || tID != targetID {
				t.Error("unexpected IDs passed to RemoveMember")
			}
			return nil
		},
	}

	h := NewMemberHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/members/"+targetID.String(), nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, callerID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", targetID.String())
	rr := httptest.NewRecorder()

	h.Remove(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestMemberRemove_NotFound(t *testing.T) {
	orgID := uuid.New()
	callerID := uuid.New()
	targetID := uuid.New()
	mock := &mockMemberService{
		removeMemberFn: func(ctx context.Context, oID, cID, tID uuid.UUID) error {
			return &service.NotFoundError{Resource: "member", Message: "user is not a member"}
		},
	}

	h := NewMemberHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/members/"+targetID.String(), nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, callerID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", targetID.String())
	rr := httptest.NewRecorder()

	h.Remove(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
