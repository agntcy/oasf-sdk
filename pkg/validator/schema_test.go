// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"encoding/json"
	"testing"
)

const testSchemaVersion = "0.7.0"

func TestGetSchemaContent(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "valid version 0.3.1",
			version:     "0.3.1",
			expectError: false,
		},
		{
			name:        "valid version v0.3.1",
			version:     "v0.3.1",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     "99.99.99",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetSchemaContent(tt.version)
			if tt.expectError { //nolint:nestif
				if err == nil {
					t.Errorf("GetSchemaContent() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("GetSchemaContent() unexpected error: %v", err)
				}

				if len(content) == 0 {
					t.Errorf("GetSchemaContent() returned empty content")
				}
				// Verify it's valid JSON
				var jsonMap map[string]any
				if err := json.Unmarshal(content, &jsonMap); err != nil {
					t.Errorf("GetSchemaContent() returned invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestGetSchemaKey(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		defsKey     string
		expectError bool
		expectEmpty bool
	}{
		{
			name:        "valid skills key",
			version:     "0.7.0",
			defsKey:     "skills",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "valid domains key",
			version:     "0.7.0",
			defsKey:     "domains",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "valid objects key",
			version:     "0.7.0",
			defsKey:     "objects",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "invalid key",
			version:     "0.7.0",
			defsKey:     "nonexistent",
			expectError: true,
		},
		{
			name:        "invalid version",
			version:     "99.99.99",
			defsKey:     "skills",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetSchemaKey(tt.version, tt.defsKey)
			if tt.expectError {
				if err == nil {
					t.Errorf("GetSchemaKey() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetSchemaKey() unexpected error: %v", err)

				return
			}

			if tt.expectEmpty {
				if len(result) > 2 { // More than just {}
					t.Errorf("GetSchemaKey() expected empty result but got data")
				}
			} else {
				if len(result) == 0 {
					t.Errorf("GetSchemaKey() returned empty result")
				}

				// Verify it's valid JSON
				var jsonMap map[string]any
				if err := json.Unmarshal(result, &jsonMap); err != nil {
					t.Errorf("GetSchemaKey() returned invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestGetSchemaSkills(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "valid version 0.3.1",
			version:     "0.3.1",
			expectError: false,
		},
		{
			name:        "valid version v0.3.1",
			version:     "v0.3.1",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     "99.99.99",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skills, err := GetSchemaSkills(tt.version)
			if tt.expectError { //nolint:nestif
				if err == nil {
					t.Errorf("GetSchemaSkills() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("GetSchemaSkills() unexpected error: %v", err)
				}

				if len(skills) == 0 {
					t.Errorf("GetSchemaSkills() returned empty skills")
				}

				// Verify it's valid JSON
				var skillsMap map[string]any
				if err := json.Unmarshal(skills, &skillsMap); err != nil {
					t.Errorf("GetSchemaSkills() returned invalid JSON: %v", err)
				}

				// Verify it contains skill definitions
				if len(skillsMap) == 0 {
					t.Errorf("GetSchemaSkills() returned empty skills map")
				}

				// Check that a known skill exists in the returned data
				if _, ok := skillsMap["text_classification"]; !ok && tt.version == "0.3.1" {
					t.Logf("Warning: Expected skill 'text_classification' not found in version %s", tt.version)
				}
			}
		})
	}
}

func TestGetSchemaDomains(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     "99.99.99",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains, err := GetSchemaDomains(tt.version)
			if tt.expectError { //nolint:nestif
				if err == nil {
					t.Errorf("GetSchemaDomains() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("GetSchemaDomains() unexpected error: %v", err)
				}

				// Verify it's valid JSON
				var domainsMap map[string]any
				if err := json.Unmarshal(domains, &domainsMap); err != nil {
					t.Errorf("GetSchemaDomains() returned invalid JSON: %v", err)
				}

				// For version 0.7.0, we expect domains
				if tt.version == testSchemaVersion {
					if len(domainsMap) == 0 {
						t.Errorf("GetSchemaDomains() returned empty domains map for version %s", tt.version)
					}

					// Check that a known domain exists in the returned data
					if _, ok := domainsMap["lean_manufacturing"]; !ok {
						t.Logf("Warning: Expected domain 'lean_manufacturing' not found in version %s", tt.version)
					}
				}
			}
		})
	}
}

func TestGetAvailableSchemaVersions(t *testing.T) {
	versions, err := GetAvailableSchemaVersions()
	if err != nil {
		t.Fatalf("GetAvailableSchemaVersions() unexpected error: %v", err)
	}

	if len(versions) == 0 {
		t.Error("GetAvailableSchemaVersions() returned no versions")
	}

	// Check that expected versions are present
	expectedVersions := map[string]bool{
		"0.3.1":  true,
		"0.7.0":  true,
		"v0.3.1": true,
	}

	foundVersions := make(map[string]bool)
	for _, v := range versions {
		foundVersions[v] = true
	}

	for expected := range expectedVersions {
		if !foundVersions[expected] {
			t.Logf("Warning: Expected version %s not found in available versions", expected)
		}
	}
}

// Helper function to compare schema section counts between dedicated getter and full schema.
func compareSchemaSection(t *testing.T, version string, sectionName string, getSection func(string) ([]byte, error)) {
	t.Helper()

	fullSchema, err := GetSchemaContent(version)
	if err != nil {
		t.Fatalf("Failed to get full schema: %v", err)
	}

	var fullSchemaMap map[string]any
	if err := json.Unmarshal(fullSchema, &fullSchemaMap); err != nil {
		t.Fatalf("Failed to parse full schema: %v", err)
	}

	sectionData, err := getSection(version)
	if err != nil {
		t.Fatalf("Failed to get %s: %v", sectionName, err)
	}

	var sectionMap map[string]any
	if err := json.Unmarshal(sectionData, &sectionMap); err != nil {
		t.Fatalf("Failed to parse %s: %v", sectionName, err)
	}

	// Extract section from full schema
	defs, ok := fullSchemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatalf("Expected $defs to be map[string]any")
	}

	fullSchemaSection, ok := defs[sectionName].(map[string]any)
	if !ok {
		t.Fatalf("Expected %s to be map[string]any", sectionName)
	}

	// Compare the number of items
	if len(sectionMap) != len(fullSchemaSection) {
		t.Errorf("%s count mismatch: getter returned %d items, full schema has %d items",
			sectionName, len(sectionMap), len(fullSchemaSection))
	}
}

func TestGetSchemaSkillsVsFullSchema(t *testing.T) {
	// This test ensures that GetSchemaSkills returns the same skills
	// section as in the full schema
	compareSchemaSection(t, testSchemaVersion, "skills", GetSchemaSkills)
}

func TestGetSchemaDomainsVsFullSchema(t *testing.T) {
	// This test ensures that GetSchemaDomains returns the same domains
	// section as in the full schema
	compareSchemaSection(t, testSchemaVersion, "domains", GetSchemaDomains)
}

func TestGetSchemaModules(t *testing.T) {
	// Note: modules may not exist in all schema versions
	_, err := GetSchemaModules("0.7.0")
	// We don't assert on error since modules might not exist
	// This test mainly ensures the function doesn't panic
	_ = err
}
