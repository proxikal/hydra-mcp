package cli

import (
	"testing"

	"github.com/proxikal/hydra/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_ExplicitClient(t *testing.T) {
	t.Skip("Full bootstrap integration tested in bootstrap package")
}

func TestInitCommand_WizardSingleClient(t *testing.T) {
	t.Skip("Wizard flow tested via runDiscoveryWizard unit tests")
}

func TestInitCommand_WizardMultipleClients(t *testing.T) {
	t.Skip("Wizard flow tested via runDiscoveryWizard unit tests")
}

func TestInitCommand_WizardNoClients(t *testing.T) {
	t.Skip("Wizard flow tested via runDiscoveryWizard unit tests")
}

func TestBootstrapClient_UnsupportedClient(t *testing.T) {
	log := logger.New("error")
	err := bootstrapClient("invalid-client", "", "", false, log)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported client")
}

func TestSupportedClients(t *testing.T) {
	clients := supportedClients()
	assert.Contains(t, clients, "claude")
	assert.Contains(t, clients, "cline")
	assert.Contains(t, clients, "cursor")
	assert.Contains(t, clients, "claude-cli")
	assert.Contains(t, clients, "continue")
	assert.Contains(t, clients, "cursor-composer")
	assert.Len(t, clients, 6)
}
