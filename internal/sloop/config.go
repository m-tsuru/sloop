package sloop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configDirName  = ".sloop"
	configFileName = "config.yaml"
)

// Config is the repository-shared Sloop project configuration.
type Config struct {
	Project    ProjectConfig    `yaml:"project"`
	Repository RepositoryConfig `yaml:"repository"`
	Git        GitConfig        `yaml:"git"`
}

type ProjectConfig struct {
	ID         string `yaml:"id"`
	Slug       string `yaml:"slug"`
	SpecPrefix string `yaml:"spec-prefix"`
}

type RepositoryConfig struct {
	Path string `yaml:"path"`
}

type GitConfig struct {
	NotesRef string `yaml:"notes-ref"`
}

type Project struct {
	Root       string
	ConfigPath string
	Config     Config
}

func WriteConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode project config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write project config: %w", err)
	}
	return nil
}

func ReadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read project config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode project config: %w", err)
	}
	if cfg.Project.ID == "" || cfg.Project.Slug == "" || cfg.Project.SpecPrefix == "" {
		return Config{}, errors.New("project config is missing required project fields")
	}
	if cfg.Git.NotesRef == "" {
		cfg.Git.NotesRef = "refs/notes/sloop"
	}
	return cfg, nil
}

// FindProject searches cwd and its parents for .sloop/config.yaml.
func FindProject(cwd string) (Project, error) {
	current, err := filepath.Abs(cwd)
	if err != nil {
		return Project{}, fmt.Errorf("resolve current directory: %w", err)
	}
	for {
		configPath := filepath.Join(current, configDirName, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			cfg, err := ReadConfig(configPath)
			if err != nil {
				return Project{}, err
			}
			return Project{Root: current, ConfigPath: configPath, Config: cfg}, nil
		} else if !os.IsNotExist(err) {
			return Project{}, fmt.Errorf("inspect project config: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return Project{}, errors.New("not a Sloop project (no .sloop/config.yaml found)")
}
