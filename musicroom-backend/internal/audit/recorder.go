package audit

import (
	"context"
	"log"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

type Entry struct {
	UserID       uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Platform     string
	DeviceModel  string
	AppVersion   string
	IP           *netip.Addr
	UserAgent    string
	StatusCode   int
}

type Recorder struct {
	q    *repository.Queries
	ch   chan Entry
	done chan struct{}
}

func NewRecorder(q *repository.Queries, buffer int) *Recorder {
	r := &Recorder{
		q:    q,
		ch:   make(chan Entry, buffer),
		done: make(chan struct{}),
	}
	go r.worker()
	return r
}

func (r *Recorder) Record(e Entry) {
	select {
	case r.ch <- e:
	default:
		log.Printf("audit: buffer full, log dropped (action=%q)", e.Action)
	}
}

func (r *Recorder) worker() {
	for e := range r.ch {
		r.write(e)
	}
	close(r.done)
}

func (r *Recorder) write(e Entry) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var uid pgtype.UUID
	if e.UserID != uuid.Nil {
		uid = pgtype.UUID{Bytes: e.UserID, Valid: true}
	}
	var status *int32
	if e.StatusCode != 0 {
		s := int32(e.StatusCode)
		status = &s
	}
	if err := r.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserID:       uid,
		Action:       e.Action,
		ResourceType: ptr(e.ResourceType),
		ResourceID:   ptr(e.ResourceID),
		Platform:     ptr(e.Platform),
		DeviceModel:  ptr(e.DeviceModel),
		AppVersion:   ptr(e.AppVersion),
		IpAddress:    e.IP,
		UserAgent:    ptr(e.UserAgent),
		StatusCode:   status,
	}); err != nil {
		log.Printf("audit: write error: %v", err)
	}
}

func (r *Recorder) Close() {
	close(r.ch)
	<-r.done
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
