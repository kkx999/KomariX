package migrations

import "testing"

func TestNormalizeLegacyPluginMarketSources(t *testing.T) {
	legacy := pluginMarketSourceMigration{
		ID: "official", Name: "Komari Official",
		URL: legacyOfficialPluginMarketURL, Enabled: true,
	}
	thirdParty := pluginMarketSourceMigration{
		ID: "custom", Name: "Custom",
		URL: "https://example.com/plugins.json", Enabled: true,
	}

	got, changed := normalizeLegacyPluginMarketSources([]pluginMarketSourceMigration{legacy, thirdParty})
	if !changed {
		t.Fatal("expected legacy official source to be migrated")
	}
	if len(got) != 2 {
		t.Fatalf("got %d sources, want 2", len(got))
	}
	if got[0].ID != "official" || got[0].Name != "KomariX 市场镜像" || got[0].URL != komarixOfficialPluginMarketURL || !got[0].Enabled {
		t.Fatalf("migrated official source = %#v", got[0])
	}
	if got[1] != thirdParty {
		t.Fatalf("third-party source changed: %#v", got[1])
	}
}

func TestNormalizeLegacyPluginMarketSourcesAvoidsDuplicateOfficial(t *testing.T) {
	sources := []pluginMarketSourceMigration{
		{ID: "legacy", Name: "Komari Official", URL: legacyOfficialPluginMarketURL, Enabled: true},
		{ID: "official", Name: "KomariX 市场镜像", URL: komarixOfficialPluginMarketURL, Enabled: true},
		{ID: "custom", Name: "Custom", URL: "https://example.com/plugins.json", Enabled: false},
	}
	got, changed := normalizeLegacyPluginMarketSources(sources)
	if !changed {
		t.Fatal("expected duplicate legacy source to be removed")
	}
	if len(got) != 2 {
		t.Fatalf("got %d sources, want 2: %#v", len(got), got)
	}
	for _, source := range got {
		if source.URL == legacyOfficialPluginMarketURL {
			t.Fatalf("legacy source remained: %#v", got)
		}
	}
}

func TestNormalizeLegacyPluginMarketSourcesLeavesCustomSourcesAlone(t *testing.T) {
	sources := []pluginMarketSourceMigration{
		{ID: "custom", Name: "Custom", URL: "https://example.com/plugins.json", Enabled: true},
	}
	got, changed := normalizeLegacyPluginMarketSources(sources)
	if changed {
		t.Fatal("custom-only source list should not be changed")
	}
	if len(got) != 1 || got[0] != sources[0] {
		t.Fatalf("custom source changed: %#v", got)
	}
}
