package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type Env struct {
	Id            string            `yaml:"id"`
	Name          string            `yaml:"name"`
	Default       bool              `yaml:"default"`
	Vars          map[string]string `yaml:"vars"`
	Notifications []Notification    `yaml:"notifications"`
	Checks        []Check
}

type Notification struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

type Check struct {
	Id      string            `yaml:"id"`
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Every   time.Duration     `yaml:"every"`
	Timeout time.Duration     `yaml:"timeout"`
	Envs    []string          `yaml:"envs"`
	Params  map[string]string `yaml:"params"`
}

type Config struct {
	DBPath       string
	Listen       string
	Static       string
	Worker       int
	Duration     time.Duration
	Environments []Env
}

func getConfig() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.DBPath, "db", "insantus.db", "Path to the sqlite storage db file")
	flag.StringVar(&cfg.Listen, "listen", ":8080", "Server and port to listen")
	flag.StringVar(&cfg.Static, "static", "static", "Directory with static content to serve")
	flag.IntVar(&cfg.Worker, "worker", 20, "Number of cheks to run in parallel")
	flag.DurationVar(&cfg.Duration, "duration", time.Minute, "Default duration for the checks")

	var checksPath, environmentsPath string
	flag.StringVar(&environmentsPath, "environments", "environments.yml", "The YAML config for the environments")
	flag.StringVar(&checksPath, "checks", "checks.yml", "The YAML config fot the checks")
	flag.Parse()

	var err error
	cfg.Environments, err = readEnvironments(environmentsPath)
	if err != nil {
		return nil, err
	}

	for i, e := range cfg.Environments {
		allChecks, err := readChecksForEnvironment(checksPath, e)
		if err != nil {
			return nil, err
		}
		cfg.Environments[i].Checks = []Check{}
		for _, c := range allChecks {
			if contains(c.Envs, e.Id) {
				cfg.Environments[i].Checks = append(cfg.Environments[i].Checks, c)
			}
		}
	}

	return cfg, nil
}

func readEnvironments(environmentsPath string) ([]Env, error) {
	b, err := ioutil.ReadFile(environmentsPath)
	if err != nil {
		return nil, err
	}
	b = []byte(os.Expand(string(b), func(varName string) string {
		if s, envExist := os.LookupEnv(varName); envExist {
			return strings.Replace(s, "\n", "\\n", -1)
		}
		return ""
	}))

	envs := []Env{}
	return envs, yaml.Unmarshal(b, &envs)
}

func readChecksForEnvironment(checksPath string, e Env) ([]Check, error) {
	b, err := ioutil.ReadFile(checksPath)
	if err != nil {
		return nil, err
	}
	b = []byte(os.Expand(string(b), func(varName string) string {
		if s, envExist := os.LookupEnv(varName); envExist {
			return strings.Replace(s, "\n", "\\n", -1)
		}
		return strings.Replace(e.Vars[varName], "\n", "\\n", -1)
	}))
	checks := []Check{}
	return checks, yaml.Unmarshal(b, &checks)
}
