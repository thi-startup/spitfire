package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/thi-startup/spitfire/utils"
)

type InitConfig struct {
	ImageConfig ImageConfig
	CmdOverride []string
	RootDevice  string
	TTY         bool
	Hostname    string
	ExtraEnv    []string
	Mounts      []Mounts
	EtcResolv   EtcResolv
	EtcHost     []EtcHost
}

type ImageConfig struct {
	Cmd        []string
	Entrypoint []string
	Env        []string
	WorkingDir string
	User       string
}

type Mounts struct {
	MountPath  string
	DevicePath string
}

type EtcResolv struct {
	Nameservers []string
}

type EtcHost struct {
	Host string
	IP   string
	Desc string
}

func NewInitFromImageConfig(config *v1.Config) *InitConfig {
	return &InitConfig{
		ImageConfig: ImageConfig{
			Cmd:        config.Cmd,
			Entrypoint: config.Entrypoint,
			Env:        config.Env,
			WorkingDir: config.WorkingDir,
			User:       config.User,
		},
		EtcResolv: EtcResolv{
			Nameservers: []string{"8.8.8.8", "8.8.4.4"},
		},
		ExtraEnv:   []string{"TERM=xterm"},
		Hostname:   "localhost",
		RootDevice: "/dev/vdb",
	}
}

func WriteInitConfig(destDir string, config *InitConfig) error {
	b, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling struct: %w", err)
	}

	if !utils.IsDir(destDir) {
		return fmt.Errorf("error destination must be a directory: %w", err)
	}

	cfg := filepath.Join(destDir, "run.json")

	if err := os.WriteFile(cfg, b, os.ModePerm); err != nil {
		return fmt.Errorf("error writing config to file: %w", err)
	}

	return nil
}

func ReadInitConfig(path string) (*InitConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer f.Close()

	var cfg InitConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding json config: %w", err)
	}

	return &cfg, err
}
