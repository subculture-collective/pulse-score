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
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockAlertRuleService struct {
	listFn   func(ctx context.Context, orgID uuid.UUID) ([]*repository.AlertRule, error)
	getByIDFn func(ctx context.Context, id, orgID uuid.UUID) (*repository.AlertRule, error)
	createFn func(ctx context.Context, orgID, userID uuid.UUID, req service.CreateAlertRuleRequest) (*repository.AlertRule, error)
	updateFn func(ctx context.Context, id, orgID uuid.UUID, req service.UpdateAlertRuleRequest) (*repository.AlertRule, error)
	deleteFn func(ctx context.Context, id, orgID uuid.UUID) error
}

func (m *mockAlertRuleService) List(ctx context.Context, orgID uuid.UUID) ([]*repository.AlertRule, error) {
	return m.listFn(ctx, orgID)
}

func (m *mockAlertRuleService) GetByID(ctx context.Context, id, orgID uuid.UUID) (*repository.AlertRule, error) {
	return m.getByIDFn(ctx, id, orgID)
}

func (m *mockAlertRuleService) Create(ctx context.Context, orgID, userID uuid.UUID, req service.CreateAlertRuleRequest) (*repository.AlertRule, error) {
	return m.createFn(ctx, orgID, userID, req)
}

func (m *mockAlertRuleService) Update(ctx context.Context, id, orgID uuid.UUID, req service.UpdateAlertRuleRequest) (*repository.AlertRule, error) {
	return m.updateFn(ctx, id, orgID, req)
}

func (m *mockAlertRuleService) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return m.deleteFn(ctx, id, orgID)
}

func TestAlertRuleList_Unauthorized(t *testing.T) {
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleList_Success(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		listFn: func(ctx context.Context, oID uuid.UUID) ([]*repository.AlertRule, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return []*repository.AlertRule{
				{ID: ruleID, Name: "score drop alert"},
			}, nil
		},
	}

	h := NewAlertRuleHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAlertRuleGet_Unauthorized(t *testing.T) {
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules/"+uuid.New().String(), nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleGet_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules/bad", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "bad")
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAlertRuleGet_NotFound(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		getByIDFn: func(ctx context.Context, id, oID uuid.UUID) (*repository.AlertRule, error) {
			return nil, &service.NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
		},
	}

	h := NewAlertRuleHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules/"+ruleID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestAlertRuleGet_Success(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		getByIDFn: func(ctx context.Context, id, oID uuid.UUID) (*repository.AlertRule, error) {
			if id != ruleID {
				t.Errorf("expected ruleID %s, got %s", ruleID, id)
			}
			return &repository.AlertRule{ID: ruleID, Name: "test rule"}, nil
		},
	}

	h := NewAlertRuleHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rules/"+ruleID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAlertRuleCreate_Unauthorized_NoOrgID(t *testing.T) {
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", nil)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleCreate_Unauthorized_NoUserID(t *testing.T) {
	orgID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleCreate_InvalidBody(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", bytes.NewBufferString("{bad"))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAlertRuleCreate_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		createFn: func(ctx context.Context, oID, uID uuid.UUID, req service.CreateAlertRuleRequest) (*repository.AlertRule, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			if uID != userID {
				t.Errorf("expected userID %s, got %s", userID, uID)
			}
			if req.Name != "Score Drop" {
				t.Errorf("expected name 'Score Drop', got %s", req.Name)
			}
			return &repository.AlertRule{ID: ruleID, Name: "Score Drop"}, nil
		},
	}

	h := NewAlertRuleHandler(mock)
	body, _ := json.Marshal(map[string]any{
		"name":         "Score Drop",
		"trigger_type": "score_drop",
		"channel":      "email",
		"recipients":   []string{"admin@example.com"},
		"conditions":   map[string]any{"threshold": 50},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", bytes.NewReader(body))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestAlertRuleCreate_ValidationError(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	mock := &mockAlertRuleService{
		createFn: func(ctx context.Context, oID, uID uuid.UUID, req service.CreateAlertRuleRequest) (*repository.AlertRule, error) {
			return nil, &service.ValidationError{Field: "name", Message: "name is required"}
		},
	}

	h := NewAlertRuleHandler(mock)
	body, _ := json.Marshal(map[string]any{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rules", bytes.NewReader(body))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestAlertRuleUpdate_Unauthorized(t *testing.T) {
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/rules/"+uuid.New().String(), nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleUpdate_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/rules/bad", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "bad")
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAlertRuleUpdate_InvalidBody(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/rules/"+ruleID.String(), bytes.NewBufferString("not json"))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAlertRuleUpdate_Success(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	newName := "Updated Rule"
	mock := &mockAlertRuleService{
		updateFn: func(ctx context.Context, id, oID uuid.UUID, req service.UpdateAlertRuleRequest) (*repository.AlertRule, error) {
			if id != ruleID {
				t.Errorf("expected ruleID %s, got %s", ruleID, id)
			}
			return &repository.AlertRule{ID: ruleID, Name: newName}, nil
		},
	}

	h := NewAlertRuleHandler(mock)
	body, _ := json.Marshal(map[string]any{"name": newName})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/rules/"+ruleID.String(), bytes.NewReader(body))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAlertRuleDelete_Unauthorized(t *testing.T) {
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/rules/"+uuid.New().String(), nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAlertRuleDelete_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	h := NewAlertRuleHandler(&mockAlertRuleService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/rules/bad", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "bad")
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAlertRuleDelete_Success(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		deleteFn: func(ctx context.Context, id, oID uuid.UUID) error {
			if id != ruleID {
				t.Errorf("expected ruleID %s, got %s", ruleID, id)
			}
			return nil
		},
	}

	h := NewAlertRuleHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/rules/"+ruleID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestAlertRuleDelete_NotFound(t *testing.T) {
	orgID := uuid.New()
	ruleID := uuid.New()
	mock := &mockAlertRuleService{
		deleteFn: func(ctx context.Context, id, oID uuid.UUID) error {
			return &service.NotFoundError{Resource: "alert_rule", Message: "alert rule not found"}
		},
	}

	h := NewAlertRuleHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/rules/"+ruleID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", ruleID.String())
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
