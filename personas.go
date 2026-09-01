package scambus

import "context"

type PersonaService struct{ client *Client }

func (s *PersonaService) List(ctx context.Context) ([]Persona, error) {
	var out []Persona
	if err := s.client.get(ctx, "/personas", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PersonaService) Get(ctx context.Context, personaID string) (*Persona, error) {
	var out Persona
	if err := s.client.get(ctx, "/personas/"+personaID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PersonaIdentifierInput struct {
	IdentifierID string `json:"identifier_id,omitempty"`
	Type         string `json:"type,omitempty"`
	Value        string `json:"value,omitempty"`
	Annotation   string `json:"annotation,omitempty"`
}

type CreatePersonaInput struct {
	Name              string                   `json:"name"`
	Description       string                   `json:"description,omitempty"`
	Personality       string                   `json:"personality,omitempty"`
	Background        string                   `json:"background,omitempty"`
	AddressLine1      string                   `json:"address_line1,omitempty"`
	AddressLine2      string                   `json:"address_line2,omitempty"`
	AddressCity       string                   `json:"address_city,omitempty"`
	AddressState      string                   `json:"address_state,omitempty"`
	AddressPostalCode string                   `json:"address_postal_code,omitempty"`
	AddressCountry    string                   `json:"address_country,omitempty"`
	Identifiers       []PersonaIdentifierInput `json:"identifiers,omitempty"`
}

func (s *PersonaService) Create(ctx context.Context, in CreatePersonaInput) (*Persona, error) {
	var out Persona
	if err := s.client.post(ctx, "/personas", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdatePersonaInput struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	Personality       *string `json:"personality,omitempty"`
	Background        *string `json:"background,omitempty"`
	AddressLine1      *string `json:"address_line1,omitempty"`
	AddressLine2      *string `json:"address_line2,omitempty"`
	AddressCity       *string `json:"address_city,omitempty"`
	AddressState      *string `json:"address_state,omitempty"`
	AddressPostalCode *string `json:"address_postal_code,omitempty"`
	AddressCountry    *string `json:"address_country,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

func (s *PersonaService) Update(ctx context.Context, personaID string, in UpdatePersonaInput) (*Persona, error) {
	var out Persona
	if err := s.client.put(ctx, "/personas/"+personaID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *PersonaService) Delete(ctx context.Context, personaID string) error {
	return s.client.delete(ctx, "/personas/"+personaID)
}

type PersonaMediaInput struct {
	MediaID  string `json:"media_id"`
	Category string `json:"category"`
	Notes    string `json:"notes"`
}

func (s *PersonaService) AddMedia(ctx context.Context, personaID string, in PersonaMediaInput) (*PersonaMediaLink, error) {
	if in.Category == "" {
		in.Category = "other"
	}
	var out PersonaMediaLink
	if err := s.client.post(ctx, "/personas/"+personaID+"/media", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdatePersonaMediaInput struct {
	Category *string `json:"category,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

func (s *PersonaService) UpdateMedia(ctx context.Context, personaID, mediaID string, in UpdatePersonaMediaInput) error {
	return s.client.put(ctx, "/personas/"+personaID+"/media/"+mediaID, in, nil)
}

func (s *PersonaService) RemoveMedia(ctx context.Context, personaID, mediaID string) error {
	return s.client.delete(ctx, "/personas/"+personaID+"/media/"+mediaID)
}
