package yas3trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/tekig/photo-backup-server/internal/entity"
	"github.com/tekig/photo-backup-server/internal/photo"
)

type HTTP struct {
	serve http.Server
	photo *photo.Photo
}

type ConfigHTTP struct {
	Listen string
	Photo  *photo.Photo
	Logger *slog.Logger
}

func NewHTTP(config ConfigHTTP) (*HTTP, error) {
	gateway := &HTTP{
		serve: http.Server{
			Addr: config.Listen,
		},
		photo: config.Photo,
	}

	gateway.serve.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := gateway.handler(r, w); err != nil {
			config.Logger.Error(
				"ya-s3-trigger handler",
				slog.String("error", err.Error()),
			)
		}
	})

	return gateway, nil
}

func (g *HTTP) Run() error {
	return g.serve.ListenAndServe()
}

func (g *HTTP) Shutdown() error {
	return g.serve.Shutdown(context.Background())
}

//	{
//		"messages": [
//		  {
//			"event_metadata": {
//			  "event_id": "bb1dd06d-a82c-49b4-af98-d8e0********",
//			  "event_type": "yandex.cloud.events.storage.ObjectDelete",
//			  "created_at": "2019-12-19T14:17:47.847365Z",
//			  "tracing_context": {
//				"trace_id": "dd52ace7********",
//				"span_id": "",
//				"parent_span_id": ""
//			  },
//			  "cloud_id": "b1gvlrnlei4l********",
//			  "folder_id": "b1g88tflru0e********"
//			},
//			"details": {
//			  "bucket_id": "s3-for-trigger",
//			  "object_id": "dev/0_15a775_972dbde4_orig12.jpg"
//			}
//		  }
//		]
//	}
type Request struct {
	Messages []Message `json:"messages,omitempty"`
}

type Message struct {
	Event  Event  `json:"event_metadata,omitempty"`
	Object Object `json:"details,omitempty"`
}

type Event struct {
	EventType string `json:"event_type,omitempty"`
}

type Object struct {
	BucketID string `json:"bucket_id,omitempty"`
	ObjectID string `json:"object_id,omitempty"`
}

func (g *HTTP) handler(r *http.Request, w http.ResponseWriter) error {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	var events = make([]entity.Event, 0, len(req.Messages))
	for _, msg := range req.Messages {
		var eventType entity.EventType
		switch msg.Event.EventType {
		case "yandex.cloud.events.storage.ObjectDelete":
			eventType = entity.EventTypeObjectDelete
		case "yandex.cloud.events.storage.ObjectCreate":
			eventType = entity.EventTypeObjectCreate
		default:
			return fmt.Errorf("unknown event type %s", msg.Event.EventType)
		}

		events = append(events, entity.Event{
			BacketID:  msg.Object.BucketID,
			ObjectID:  msg.Object.ObjectID,
			EventType: eventType,
		})
	}

	if err := g.photo.Events(r.Context(), events); err != nil {
		return fmt.Errorf("events: %w", err)
	}

	w.WriteHeader(http.StatusOK)

	return nil
}
