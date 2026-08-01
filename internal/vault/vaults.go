package vault

import "fmt"

// VaultCatalog is the ordered set of vaults in the effective knowledge view.
type VaultCatalog struct {
	Vaults []Origin `json:"vaults"`
}

// Vaults returns configured vaults in precedence order followed by the bundle.
func Vaults(root string) (VaultCatalog, error) {
	effective, err := loadEffectiveVault(root)
	if err != nil {
		return VaultCatalog{}, fmt.Errorf("vaults: %w", err)
	}

	entries := make([]Origin, 0, len(effective.sources)+1)
	hasCore := false
	for precedence, source := range effective.sources {
		if source.config.Vault.Name == "core" {
			hasCore = true
		}
		entries = append(entries, source.originAt("", precedence))
	}
	if !hasCore {
		entries = append(entries, Origin{
			Vault:      "core",
			Kind:       OriginBundle,
			Precedence: len(entries),
		})
	}
	return VaultCatalog{Vaults: entries}, nil
}
