package scambus

import "context"

type AutomationService struct{ client *Client }

func (s *AutomationService) List(ctx context.Context) ([]Automation, error) {
	var out []Automation
	if err := s.client.get(ctx, "/automations", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
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
	Active      bool   `json:"active"`
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
		body["expiresAt"] = expiresAt
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
