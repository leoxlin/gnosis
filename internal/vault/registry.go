package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type vaultRegistration struct {
	name     string
	target   string
	identity string
	source   string
}

// ResolveTarget resolves an optional configured vault name from the nearest
// local configuration and the user-level registry.
func ResolveTarget(start, selected string) (string, error) {
	localPath, err := findConfigPath(start)
	if err != nil {
		return "", err
	}
	userPath, err := userConfigPath()
	if err != nil {
		return "", err
	}

	var registrations []vaultRegistration
	var localDefault string
	if localPath != "" {
		config, err := loadConfigPath(localPath)
		if err != nil {
			return "", err
		}
		registrations, err = appendRegistrations(registrations, config, localPath)
		if err != nil {
			return "", err
		}
		if config.HasLocalVault() {
			localDefault = config.Vault.Name
		}
	}
	if filepath.Clean(userPath) != filepath.Clean(localPath) {
		if info, statErr := os.Stat(userPath); statErr == nil && !info.IsDir() {
			config, err := loadConfigPath(userPath)
			if err != nil {
				return "", err
			}
			registrations, err = appendRegistrations(registrations, config, userPath)
			if err != nil {
				return "", err
			}
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return "", fmt.Errorf("stat %s: %w", userPath, statErr)
		}
	}

	registry := make(map[string]vaultRegistration, len(registrations))
	for _, registration := range registrations {
		existing, ok := registry[registration.name]
		if !ok {
			registry[registration.name] = registration
			continue
		}
		if existing.identity != registration.identity {
			return "", fmt.Errorf(
				"vault name %q conflicts between %s and %s",
				registration.name,
				existing.source,
				registration.source,
			)
		}
	}

	if selected == "" {
		if localDefault == "" {
			return "", fmt.Errorf("no local vault is configured; add [vault] to the nearest gnosis.toml or select a configured vault with --vault")
		}
		selected = localDefault
	}
	if !isCanonicalVaultName(selected) {
		return "", fmt.Errorf("vault %q must be a configured canonical vault name; configured vaults: %s", selected, configuredNames(registry))
	}
	registration, ok := registry[selected]
	if !ok {
		return "", fmt.Errorf("vault %q is not configured; configured vaults: %s", selected, configuredNames(registry))
	}
	return registration.target, nil
}

func appendRegistrations(registrations []vaultRegistration, config Config, configPath string) ([]vaultRegistration, error) {
	root := filepath.Dir(configPath)
	if config.HasLocalVault() {
		identity, err := localTargetIdentity(root)
		if err != nil {
			return nil, fmt.Errorf("resolve %s [vault]: %w", configPath, err)
		}
		registrations = append(registrations, vaultRegistration{
			name:     config.Vault.Name,
			target:   root,
			identity: identity,
			source:   configPath + " [vault]",
		})
	}
	for index, declared := range config.Vaults {
		target, identity, err := registrationTarget(declared.Root, root)
		if err != nil {
			return nil, fmt.Errorf("resolve %s vaults[%d]: %w", configPath, index, err)
		}
		registrations = append(registrations, vaultRegistration{
			name:     declared.Name,
			target:   target,
			identity: identity,
			source:   fmt.Sprintf("%s vaults[%d]", configPath, index),
		})
	}
	return registrations, nil
}

func registrationTarget(value, root string) (string, string, error) {
	remote, ok, err := parseRemoteLocator(strings.TrimSpace(value))
	if err != nil {
		return "", "", err
	}
	if ok {
		return remote, "remote:" + remote, nil
	}
	target, err := resolveLocalDeclaredVaultRoot(value, root)
	if err != nil {
		return "", "", err
	}
	identity, err := localTargetIdentity(target)
	if err != nil {
		return "", "", err
	}
	return target, identity, nil
}

func localTargetIdentity(target string) (string, error) {
	target, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return "local:" + filepath.Clean(target), nil
}

func configuredNames(registry map[string]vaultRegistration) string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
