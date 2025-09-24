package main

import (
	"fmt"
	"log"

	"golang-yaml/v1"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Features []string       `yaml:"features"`
	Debug    bool           `yaml:"debug"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	TLS  bool   `yaml:"tls"`
}

type DatabaseConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Name string `yaml:"name"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

func main() {
	yamlData := `
server:
  host: localhost
  port: 8080
  tls: true

database:
  host: db.example.com
  port: 5432
  name: myapp
  user: admin
  pass: secret123

features:
  - auth
  - api
  - metrics

debug: true
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("Parsed config: %+v\n", config)

	output, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("\nMarshaled YAML:\n%s", string(output))
}
