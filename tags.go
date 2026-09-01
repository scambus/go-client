package scambus

import "context"

type TagService struct{ client *Client }

type ListTagsResult struct {
	Data       []Tag      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

func (s *TagService) List(ctx context.Context, page *PageRequest) (*ListTagsResult, error) {
	var out ListTagsResult
	if err := s.client.get(ctx, "/tags", page.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *TagService) Get(ctx context.Context, tagID string) (*Tag, error) {
	var out Tag
	if err := s.client.get(ctx, "/tags/"+tagID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateTagInput struct {
	Title              string           `json:"title"`
	TagType            string           `json:"tag_type"`
	Description        string           `json:"description,omitempty"`
	Aliases            []string         `json:"aliases,omitempty"`
	IsGlobal           bool             `json:"is_global,omitempty"`
	FlowUp             bool             `json:"flow_up"`
	FlowDown           bool             `json:"flow_down"`
	AllowDynamicValues bool             `json:"allow_dynamic_values"`
	Color              string           `json:"color,omitempty"`
	Icon               string           `json:"icon,omitempty"`
	AllocatesKarma     *int             `json:"allocates_karma,omitempty"`
	Metadata           map[string]any   `json:"metadata,omitempty"`
	TagValues          []CreateTagValue `json:"tag_values,omitempty"`
}

type CreateTagValue struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order,omitempty"`
}

func (s *TagService) Create(ctx context.Context, in CreateTagInput) (*Tag, error) {
	if in.TagType == "" {
		in.TagType = "valued"
	}
	var out Tag
	if err := s.client.post(ctx, "/tags", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateTagInput struct {
	Title              *string          `json:"title,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Aliases            []string         `json:"aliases,omitempty"`
	IsGlobal           *bool            `json:"is_global,omitempty"`
	FlowUp             *bool            `json:"flow_up,omitempty"`
	FlowDown           *bool            `json:"flow_down,omitempty"`
	AllowDynamicValues *bool            `json:"allow_dynamic_values,omitempty"`
	Color              *string          `json:"color,omitempty"`
	Icon               *string          `json:"icon,omitempty"`
	Active             *bool            `json:"active,omitempty"`
	TagValues          []CreateTagValue `json:"tag_values,omitempty"`
}

func (s *TagService) Update(ctx context.Context, tagID string, in UpdateTagInput) (*Tag, error) {
	var out Tag
	if err := s.client.put(ctx, "/tags/"+tagID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *TagService) Delete(ctx context.Context, tagID string) error {
	return s.client.delete(ctx, "/tags/"+tagID)
}

func (s *TagService) ListValues(ctx context.Context, tagID string) ([]TagValue, error) {
	var out []TagValue
	if err := s.client.get(ctx, "/tags/"+tagID+"/values", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TagService) CreateValue(ctx context.Context, tagID string, in CreateTagValue) (*TagValue, error) {
	var out TagValue
	if err := s.client.post(ctx, "/tags/"+tagID+"/values", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateTagValueInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Order       *int    `json:"order,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

func (s *TagService) UpdateValue(ctx context.Context, tagID, valueID string, in UpdateTagValueInput) (*TagValue, error) {
	var out TagValue
	if err := s.client.put(ctx, "/tags/"+tagID+"/values/"+valueID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *TagService) DeleteValue(ctx context.Context, tagID, valueID string) error {
	return s.client.delete(ctx, "/tags/"+tagID+"/values/"+valueID)
}

func (s *TagService) Effective(ctx context.Context, entityType, entityID string) ([]map[string]any, error) {
	var out []map[string]any
	if err := s.client.get(ctx, "/tags/effective/"+entityType+"/"+entityID, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
