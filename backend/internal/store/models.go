package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	UUID        uuid.UUID      `bun:"type:uuid,pk,default:gen_random_uuid()"`
	ID          int64          `bun:",autoincrement"`
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt   bun.NullTime   `bun:",soft_delete,nullzero"`
	UserID      string         `bun:",unique,notnull"`
	Email       string         `bun:""`
	FirstName   string         `bun:""`
	LastName    string         `bun:""`
	ProjectUUID uuid.UUID      `bun:"type:uuid,notnull"`
	Metadata    map[string]any `bun:"type:jsonb,nullzero"`
}

type Session struct {
	bun.BaseModel `bun:"table:sessions,alias:s"`

	UUID        uuid.UUID      `bun:"type:uuid,pk,default:gen_random_uuid()"`
	ID          int64          `bun:",autoincrement"`
	SessionID   string         `bun:",unique,notnull"`
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	DeletedAt   bun.NullTime   `bun:",soft_delete,nullzero"`
	EndedAt     *time.Time     `bun:",nullzero"`
	Metadata    map[string]any `bun:"type:jsonb,nullzero"`
	UserID      *string        `bun:""`
	ProjectUUID uuid.UUID      `bun:"type:uuid,notnull"`
}

type Message struct {
	bun.BaseModel `bun:"table:messages,alias:m"`

	UUID        uuid.UUID      `bun:"type:uuid,pk,default:gen_random_uuid()"`
	ID          int64          `bun:",autoincrement"`
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time      `bun:",nullzero"`
	DeletedAt   bun.NullTime   `bun:",soft_delete,nullzero"`
	SessionID   string         `bun:",notnull"`
	ProjectUUID uuid.UUID      `bun:"type:uuid,notnull"`
	Role        string         `bun:",notnull"`
	RoleType    string         `bun:"type:public.role_type_enum,notnull,default:'norole'"`
	Content     string         `bun:",notnull"`
	TokenCount  int            `bun:",notnull,default:0"`
	Metadata    map[string]any `bun:"type:jsonb,nullzero"`
	Name        string         `bun:""`
	Processed   *bool          `bun:""`
}

type ContextTemplate struct {
	bun.BaseModel `bun:"table:context_templates,alias:ct"`

	UUID        uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()"`
	ID          string    `bun:",unique,notnull"` // external template id
	Name        string    `bun:",notnull"`
	Content     string    `bun:",notnull"`
	ProjectUUID uuid.UUID `bun:"type:uuid,notnull"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type GraphRecord struct {
	bun.BaseModel `bun:"table:graphs,alias:g"`

	UUID        uuid.UUID      `bun:"type:uuid,pk,default:gen_random_uuid()"`
	GraphID     string         `bun:",unique,notnull"`
	UserID      *string        `bun:""`
	ProjectUUID uuid.UUID      `bun:"type:uuid,notnull"`
	Metadata    map[string]any `bun:"type:jsonb,nullzero"`
	CreatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
}

type TaskRecord struct {
	bun.BaseModel `bun:"table:tasks,alias:t"`

	UUID        uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()"`
	TaskID      string    `bun:",unique,notnull"`
	Status      string    `bun:",notnull"`
	Progress    float64   `bun:",notnull,default:0"`
	Error       string    `bun:""`
	ProjectUUID uuid.UUID `bun:"type:uuid,notnull"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type CustomInstructionRow struct {
	bun.BaseModel `bun:"table:custom_instructions,alias:ci"`

	UUID        uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()"`
	Name        string    `bun:",notnull"`
	Text        string    `bun:",notnull"`
	Scope       string    `bun:",notnull"` // project|user|graph
	ScopeID     string    `bun:""`         // user_id or graph_id
	ProjectUUID uuid.UUID `bun:"type:uuid,notnull"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type UserSummaryInstructionRow struct {
	bun.BaseModel `bun:"table:user_summary_instructions,alias:usi"`

	UUID        uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()"`
	Name        string    `bun:",notnull"`
	Text        string    `bun:",notnull"`
	ProjectUUID uuid.UUID `bun:"type:uuid,notnull"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type EntityTypesRow struct {
	bun.BaseModel `bun:"table:entity_types,alias:et"`

	UUID        uuid.UUID      `bun:"type:uuid,pk,default:gen_random_uuid()"`
	ProjectUUID uuid.UUID      `bun:"type:uuid,notnull"`
	Payload     map[string]any `bun:"type:jsonb,notnull"`
	UpdatedAt   time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
}

func (u *User) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		u.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (s *Session) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *Message) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		m.UpdatedAt = time.Now().UTC()
	}
	return nil
}
