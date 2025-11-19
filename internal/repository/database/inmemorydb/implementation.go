package inmemorydb

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/eurofurence/reg-payment-nexi-adapter/internal/entity"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database/dbrepo"
)

type InMemoryRepository struct {
	protocol   []*entity.ProtocolEntry
	idSequence uint32
	Now        func() time.Time
}

func Create() dbrepo.Repository {
	return &InMemoryRepository{
		Now: time.Now,
	}
}

func (r *InMemoryRepository) Open() error {
	r.protocol = make([]*entity.ProtocolEntry, 0)
	return nil
}

func (r *InMemoryRepository) Close() {
	r.protocol = nil
}

func (r *InMemoryRepository) Migrate() error {
	// nothing to do
	return nil
}

// --- log entries ---

func (r *InMemoryRepository) WriteProtocolEntry(ctx context.Context, e *entity.ProtocolEntry) error {
	newId := uint(atomic.AddUint32(&r.idSequence, 1))
	e.ID = newId

	// copy the attendee, so later modifications won't also modify it in the simulated db
	copiedEntry := *e
	copiedEntry.CreatedAt = time.Now()
	r.protocol = append(r.protocol, &copiedEntry)
	return nil
}

// --- testing ---

func (r *InMemoryRepository) ProtocolEntries() []*entity.ProtocolEntry {
	return r.protocol
}

func (r *InMemoryRepository) Clear() {
	r.protocol = make([]*entity.ProtocolEntry, 0)
	r.idSequence = 0
}
