package scambus

import (
	"context"
	"net/url"
	"strconv"
)

type NotificationService struct{ client *Client }

type ListNotificationsOptions struct {
	UnreadOnly bool
	Limit      int
	Offset     int
}

func (s *NotificationService) List(ctx context.Context, opts *ListNotificationsOptions) ([]Notification, error) {
	q := url.Values{}
	if opts != nil {
		if opts.UnreadOnly {
			q.Set("unread", "true")
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var out []Notification
	if err := s.client.get(ctx, "/notifications", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NotificationService) Get(ctx context.Context, notificationID string) (*Notification, error) {
	var out Notification
	if err := s.client.get(ctx, "/notifications/"+notificationID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *NotificationService) UnreadCount(ctx context.Context) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	if err := s.client.get(ctx, "/notifications/unread-count", nil, &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, notificationID string) error {
	return s.client.post(ctx, "/notifications/"+notificationID+"/mark-read", nil, nil)
}

func (s *NotificationService) MarkAllRead(ctx context.Context) error {
	return s.client.post(ctx, "/notifications/mark-all-read", nil, nil)
}

func (s *NotificationService) Dismiss(ctx context.Context, notificationID string) error {
	return s.client.post(ctx, "/notifications/"+notificationID+"/dismiss", nil, nil)
}

func (s *NotificationService) DismissAll(ctx context.Context) error {
	return s.client.post(ctx, "/notifications/dismiss-all", nil, nil)
}
