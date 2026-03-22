package deepagent

import "GopherAI/config"

func FeatureEnabled() bool {
	return config.GetConfig().DeepAgentConfig.Enabled
}
