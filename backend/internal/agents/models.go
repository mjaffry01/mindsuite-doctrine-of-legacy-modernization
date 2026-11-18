package agents

import "sync"

// The model of each tier can be replaced from the config file (mindsuite.json → models).
// Teams keep naming the built-in tier models; Resolve maps them to the configured ones
// when Claude is called and when results show which model was used.
var (
	modelMu  sync.RWMutex
	modelMap = map[string]string{}
)

// SetModels sets the model used for each tier. An empty value keeps the built-in model.
func SetModels(strong, balanced, fast string) {
	m := map[string]string{}
	for def, v := range map[string]string{ModelStrong: strong, ModelBalanced: balanced, ModelFast: fast} {
		if v != "" && v != def {
			m[def] = v
		}
	}
	modelMu.Lock()
	modelMap = m
	modelMu.Unlock()
}

// Resolve returns the model to use for a built-in tier model.
func Resolve(model string) string {
	modelMu.RLock()
	defer modelMu.RUnlock()
	if v, ok := modelMap[model]; ok {
		return v
	}
	return model
}

// ResolvedAgents shows each agent with the model it really uses.
func ResolvedAgents(in []AgentInfo) []AgentInfo {
	out := append([]AgentInfo(nil), in...)
	for i := range out {
		if out[i].Model != "" {
			out[i].Model = Resolve(out[i].Model)
		}
	}
	return out
}
