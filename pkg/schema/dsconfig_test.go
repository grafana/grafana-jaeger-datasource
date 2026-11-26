package schema_test

import (
	_ "embed"
	"testing"

	"github.com/grafana/dsconfig/schema"
)

//go:embed dsconfig.json
var configSchemaJSON []byte

// Mirrors the schema's jsonData keys. Test-only; the backend does not read it.
type settingsJSONModel struct {
	// Read by the backend.
	TraceIdTimeParams struct {
		Enabled bool `json:"enabled"`
	} `json:"traceIdTimeParams"`

	// Read by the Grafana server proxy, not by the plugin SDK or the backend.
	KeepCookies []string `json:"keepCookies"`

	// Read by the plugin SDK HTTP client.
	Timeout                float64 `json:"timeout"`
	OAuthPassThru          bool    `json:"oauthPassThru"`
	TLSAuth                bool    `json:"tlsAuth"`
	TLSAuthWithCACert      bool    `json:"tlsAuthWithCACert"`
	TLSSkipVerify          bool    `json:"tlsSkipVerify"`
	ServerName             string  `json:"serverName"`
	EnableSecureSocksProxy bool    `json:"enableSecureSocksProxy"`

	// Frontend-only; the backend never reads them.
	TracesToLogsV2  map[string]any `json:"tracesToLogsV2"`
	TracesToLogs    map[string]any `json:"tracesToLogs"`
	TracesToMetrics map[string]any `json:"tracesToMetrics"`
	NodeGraph       map[string]any `json:"nodeGraph"`
	SpanBar         map[string]any `json:"spanBar"`
}

//go:generate go test -run TestPlugin -generateArtifacts
func TestPlugin(t *testing.T) {
	schema.RunPluginTests(t, schema.PluginUnderTest{
		ID:                "jaeger",
		ConfigSchemaJSON:  configSchemaJSON,
		SettingsJSONModel: settingsJSONModel{},
		SecureKeys:        []string{"basicAuthPassword", "tlsCACert", "tlsClientCert", "tlsClientKey"},
	})
}
