package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

type Env struct {
	Id            string            `yaml:"id"`
	Name          string            `yaml:"name"`
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
	Worker       int
	Duration     time.Duration
	Environments []Env
}

func getConfig() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.DBPath, "db", "statuspage.db", "Path to the sqlite storage db file")
	flag.StringVar(&cfg.Listen, "listen", ":8080", "Server and port to listen")
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
		cfg.Environments[i].Checks, err = readChecksForEnvironment(checksPath, e)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func readEnvironments(environmentsPath string) ([]Env, error) {
	b, err := ioutil.ReadFile(environmentsPath)
	if err != nil {
		return nil, err
	}
	b = []byte(os.ExpandEnv(string(b)))
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
			return s
		}
		return e.Vars[varName]
	}))
	checks := []Check{}
	return checks, yaml.Unmarshal(b, &checks)
}
