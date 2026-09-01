package scambus

import "context"

type SessionService struct{ client *Client }

func (s *SessionService) List(ctx context.Context) ([]Session, error) {
	var out struct {
		Sessions []Session `json:"sessions"`
		Total    int       `json:"total"`
	}
	if err := s.client.get(ctx, "/sessions", nil, &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func (s *SessionService) Revoke(ctx context.Context, sessionID string) error {
	return s.client.post(ctx, "/sessions/"+sessionID+"/revoke", nil, nil)
}

func (s *SessionService) ListPasskeys(ctx context.Context) ([]Passkey, error) {
	var out []Passkey
	if err := s.client.get(ctx, "/passkeys", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SessionService) DeletePasskey(ctx context.Context, passkeyID string) error {
	return s.client.delete(ctx, "/passkeys/"+passkeyID)
}

func (s *SessionService) TwoFactorStatus(ctx context.Context) (*TwoFactorStatus, error) {
	var out TwoFactorStatus
	if err := s.client.get(ctx, "/passkeys/2fa", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetTwoFactor sends {"enable": …}; the server reads that key, not "enabled".
func (s *SessionService) SetTwoFactor(ctx context.Context, enable bool) (*TwoFactorResult, error) {
	var out TwoFactorResult
	if err := s.client.post(ctx, "/passkeys/2fa", map[string]bool{"enable": enable}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
