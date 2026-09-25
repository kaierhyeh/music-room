package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TicketStore struct {
	mu  sync.Mutex
	m   map[string]ticketEntry
	ttl time.Duration
}

type ticketEntry struct {
	userID  uuid.UUID
	client  Client
	expires time.Time
}

type Client struct {
	Platform    string
	DeviceModel string
	AppVersion  string
}

func NewTicketStore(ttl time.Duration) *TicketStore {
	return &TicketStore{m: make(map[string]ticketEntry), ttl: ttl}
}

func (s *TicketStore) Issue(userID uuid.UUID, client Client) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.m[tok] = ticketEntry{userID: userID, client: client, expires: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return tok, nil
}

func (s *TicketStore) Consume(tok string) (uuid.UUID, Client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[tok]
	if !ok {
		return uuid.Nil, Client{}, false
	}
	delete(s.m, tok)
	if time.Now().After(e.expires) {
		return uuid.Nil, Client{}, false
	}
	return e.userID, e.client, true
}
