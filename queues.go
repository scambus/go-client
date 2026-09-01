package scambus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type QueueService struct{ client *Client }

func (s *QueueService) List(ctx context.Context) ([]Queue, error) {
	var out []Queue
	if err := s.client.get(ctx, "/queues", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type CreateQueueInput struct {
	Name                  string         `json:"name"`
	Description           string         `json:"description,omitempty"`
	FilterCriteria        map[string]any `json:"filter_criteria,omitempty"`
	CadenceDays           *int           `json:"cadence_days,omitempty"`
	CooldownHours         *int           `json:"cooldown_hours,omitempty"`
	MaxContactsPerCluster *int           `json:"max_contacts_per_cluster,omitempty"`
	RotationEnabled       *bool          `json:"rotation_enabled,omitempty"`
	PriorityMode          string         `json:"priority_mode,omitempty"`
	AutoPopulate          *bool          `json:"auto_populate,omitempty"`
	ActorClusterID        string         `json:"actor_cluster_id,omitempty"`
}

func (s *QueueService) Create(ctx context.Context, in CreateQueueInput) (*Queue, error) {
	var out Queue
	if err := s.client.post(ctx, "/queues", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *QueueService) Get(ctx context.Context, queueID string) (*Queue, error) {
	var out Queue
	if err := s.client.get(ctx, "/queues/"+queueID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateQueueInput struct {
	Name                  *string        `json:"name,omitempty"`
	Description           *string        `json:"description,omitempty"`
	FilterCriteria        map[string]any `json:"filter_criteria,omitempty"`
	CadenceDays           *int           `json:"cadence_days,omitempty"`
	CooldownHours         *int           `json:"cooldown_hours,omitempty"`
	MaxContactsPerCluster *int           `json:"max_contacts_per_cluster,omitempty"`
	RotationEnabled       *bool          `json:"rotation_enabled,omitempty"`
	PriorityMode          *string        `json:"priority_mode,omitempty"`
	AutoPopulate          *bool          `json:"auto_populate,omitempty"`
	ActorClusterID        *string        `json:"actor_cluster_id,omitempty"`
	IsActive              *bool          `json:"is_active,omitempty"`
}

func (s *QueueService) Update(ctx context.Context, queueID string, in UpdateQueueInput) (*Queue, error) {
	var out Queue
	if err := s.client.put(ctx, "/queues/"+queueID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *QueueService) Delete(ctx context.Context, queueID string) error {
	return s.client.delete(ctx, "/queues/"+queueID)
}

func (s *QueueService) Stats(ctx context.Context, queueID string) (*QueueStats, error) {
	var out QueueStats
	if err := s.client.get(ctx, "/queues/"+queueID+"/stats", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *QueueService) ListItems(ctx context.Context, queueID, state string) ([]QueueItem, error) {
	q := url.Values{}
	if state != "" {
		q.Set("state", state)
	}
	var out []QueueItem
	if err := s.client.get(ctx, "/queues/"+queueID+"/items", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ReadStreamOptions struct {
	Cursor  string
	Limit   int
	BlockMS int
}

func (s *QueueService) ReadStream(ctx context.Context, queueID string, opts *ReadStreamOptions) (*QueueStreamResponse, error) {
	q := url.Values{"cursor": {CursorStart}}
	if opts != nil {
		if opts.Cursor != "" {
			q.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.BlockMS > 0 {
			q.Set("block_ms", strconv.Itoa(opts.BlockMS))
		}
	}
	var out QueueStreamResponse
	if err := s.client.call(ctx, request{
		method: http.MethodGet, endpoint: "/queues/" + queueID + "/stream", query: q, stream: true,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Claim returns nil when the queue holds no claimable item.
func (s *QueueService) Claim(ctx context.Context, queueID string) (*QueueItem, error) {
	var out QueueItem
	if err := s.client.post(ctx, "/queues/"+queueID+"/claim", nil, &out); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (s *QueueService) Release(ctx context.Context, queueID, itemID string) error {
	return s.client.post(ctx, s.itemPath(queueID, itemID, "release"), map[string]any{}, nil)
}

type ContactInput struct {
	ContactIdentifierID string `json:"contact_identifier_id,omitempty"`
	JournalEntryID      string `json:"journal_entry_id,omitempty"`
	Notes               string `json:"notes,omitempty"`
}

func (s *QueueService) RecordContact(ctx context.Context, queueID, itemID string, in ContactInput) error {
	return s.client.post(ctx, s.itemPath(queueID, itemID, "contact"), in, nil)
}

type ItemActionInput struct {
	Reason  string `json:"reason,omitempty"`
	Outcome string `json:"outcome,omitempty"`
	Note    string `json:"note,omitempty"`
}

func (s *QueueService) Complete(ctx context.Context, queueID, itemID string, in ItemActionInput) error {
	return s.client.post(ctx, s.itemPath(queueID, itemID, "complete"), in, nil)
}

func (s *QueueService) Drop(ctx context.Context, queueID, itemID string, in ItemActionInput) error {
	return s.client.post(ctx, s.itemPath(queueID, itemID, "drop"), in, nil)
}

type MoveItemInput struct {
	ItemActionInput
	TargetQueueID string `json:"target_queue_id"`
}

func (s *QueueService) Move(ctx context.Context, queueID, itemID string, in MoveItemInput) error {
	if in.TargetQueueID == "" {
		return fmt.Errorf("%w: target queue id is required", ErrValidation)
	}
	return s.client.post(ctx, s.itemPath(queueID, itemID, "move"), in, nil)
}

func (s *QueueService) ItemHistory(ctx context.Context, queueID, itemID string) ([]QueueContactLog, error) {
	var out []QueueContactLog
	if err := s.client.get(ctx, s.itemPath(queueID, itemID, "history"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *QueueService) ItemEvents(ctx context.Context, queueID, itemID string) ([]QueueItemEvent, error) {
	var out []QueueItemEvent
	if err := s.client.get(ctx, s.itemPath(queueID, itemID, "events"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *QueueService) ItemClusterIdentifiers(ctx context.Context, queueID, itemID, role string) ([]QueueClusterIdentifier, error) {
	if role == "" {
		role = "target"
	}
	var out []QueueClusterIdentifier
	if err := s.client.get(ctx, s.itemPath(queueID, itemID, "cluster"), url.Values{"role": {role}}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *QueueService) itemPath(queueID, itemID, action string) string {
	return "/queues/" + queueID + "/items/" + itemID + "/" + action
}
