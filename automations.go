package scambus

import "context"

type AutomationService struct{ client *Client }

type ListAutomationsResult struct {
	Data       []Automation `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

func (s *AutomationService) List(ctx context.Context, page *PageRequest) (*ListAutomationsResult, error) {
	var out ListAutomationsResult
	if err := s.client.get(ctx, "/automations", page.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AutomationService) Get(ctx context.Context, automationID string) (*Automation, error) {
	var out Automation
	if err := s.client.get(ctx, "/automations/"+automationID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateAutomationInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

func (s *AutomationService) Create(ctx context.Context, in CreateAutomationInput) (*Automation, error) {
	var out Automation
	if err := s.client.post(ctx, "/automations", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AutomationService) ListAPIKeys(ctx context.Context, automationID string) ([]AutomationAPIKey, error) {
	var out []AutomationAPIKey
	if err := s.client.get(ctx, "/automations/"+automationID+"/api-keys", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateAPIKey returns the only copy of the secret. Store it now.
func (s *AutomationService) CreateAPIKey(ctx context.Context, automationID, name, expiresAt string) (*AutomationAPIKey, error) {
	body := map[string]string{"name": name}
	if expiresAt != "" {
		body["expires_at"] = expiresAt
	}
	var out AutomationAPIKey
	if err := s.client.post(ctx, "/automations/"+automationID+"/api-keys", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AutomationService) RevokeAPIKey(ctx context.Context, automationID, keyID string) error {
	return s.client.post(ctx, "/automations/"+automationID+"/api-keys/"+keyID+"/revoke", nil, nil)
}

func (s *AutomationService) DeleteAPIKey(ctx context.Context, automationID, keyID string) error {
	return s.client.delete(ctx, "/automations/"+automationID+"/api-keys/"+keyID)
}
