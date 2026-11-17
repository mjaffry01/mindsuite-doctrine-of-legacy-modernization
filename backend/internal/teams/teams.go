// Package teams registers every agent team. Each team lives in its own folder:
//
//	batch/           batch & scheduling (10 agents)
//	code/            source code (15 agents)
//	database/        database exports (14 agents)
//	documents/       documents & diagrams (12 agents)
//	infrastructure/  environment & infrastructure inventories (13 agents)
//	messaging/       messaging & caching (12 agents)
//	network/         network edge & partners (11 agents)
//	identity/        identity & access (12 agents)
//	telemetry/       telemetry (11 agents)
//
// The shared core (run, evidence store, loader, verifier, Claude client) is internal/agents.
package teams

import (
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/batch"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/code"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/database"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/documents"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/identity"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/incidents"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/infrastructure"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/messaging"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/network"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams/telemetry"
)
