package entity

type EventType string

const (
	EventTypeObjectCreate EventType = "object_create"
	EventTypeObjectDelete EventType = "object_delete"
)

type Event struct {
	BacketID  string
	ObjectID  string
	EventType EventType
}
