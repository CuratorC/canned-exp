// Package app defines the application dependency container.
// It is intentionally a separate package to avoid circular imports
// between bootstrap and route/handler/service layers.
package app

import (
	"canned-exp/internal/experience"

	"github.com/CuratorC/gocanned/database"
)

// App holds all application-level dependencies, assembled by bootstrap
// and passed explicitly to the layers that need them.
type App struct {
	DB               *database.DB
	ExperienceService *experience.Service
}
