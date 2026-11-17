// Package teams registers every agent team. Each team lives in its own folder:
//
//	code/            source code (15 agents)
//	database/        database exports (14 agents)
//	infrastructure/  environment & infrastructure inventories (13 agents)
//	messaging/       messaging & caching (12 agents)
//	identity/        identity & access (12 agents)
//	telemetry/       telemetry (11 agents)
//
// The shared core (run, evidence store, loader, verifier, Claude client) is internal/agents.
package teams

import (
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/code"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/database"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/identity"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/infrastructure"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/messaging"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/telemetry"
)
