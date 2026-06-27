package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureUUID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&c.ID)
	return nil
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&u.ID)
	return nil
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&r.ID)
	return nil
}

func (m *ModelConfig) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&m.ID)
	return nil
}
