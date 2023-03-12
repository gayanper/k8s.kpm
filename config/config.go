package config

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type PortForwardEntry struct {
	ServiceName string `json:"serviceName,omitempty"`
	ServicePort int    `json:"servicePort,omitempty"`
	LocalPort   int    `json:"localPort,omitempty"`
}

type Config struct {
	Namespace string             `json:"namespace,omitempty"`
	Entries   []PortForwardEntry `json:"entries,omitempty"`
}

type Profile struct {
	Name          string `json:"name,omitempty"`
	Configuration Config `json:"config,omitempty"`
}

type Profiles struct {
	Profiles []Profile `json:"profiles,omitempty"`
}

// Read the configuration file from the given path and parse it and return the configured profiles.
//
// If the path is empty then then read the default configuration in the current user home directory.
// The default file location is ~/.kpm/config,json
func Read(path string) map[string]Profile {
	if path == "" {
		var home, err = os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		path = filepath.Join(home, ".kpm", "config.json")
		if !exist(path) {
			configDirPath := filepath.Join(home, ".kpm")
			if !exist(configDirPath) {
				de := os.Mkdir(configDirPath, fs.ModePerm)
				if de != nil {
					log.Fatal(de)
				}
			}

			createDefaultConfig(path)
			log.Println("new default configuration file is created at ", path)
			log.Println("please update the file before running kpm again.")
			os.Exit(1)
		}
	}

	if !exist(path) {
		log.Fatal("invalid file path : ", path)
	}

	data, de := os.ReadFile(path)
	if de != nil {
		log.Fatal(de)
	}

	var config Profiles
	ue := json.Unmarshal(data, &config)
	if ue != nil {
		log.Fatal(de)
	}

	result := make(map[string]Profile)
	for _, p := range config.Profiles {
		result[p.Name] = p
	}

	return result
}

func createDefaultConfig(path string) {
	config :=
		`{
		"profiles" : [
			{
				"name" : "default",
				"config" : 	{
					"namespace" : "default",
					"entries" : [
						{
							"serviceName" : "svc/changeme",
							"servicePort" : 80,
							"localPort" : 8080
						}
					]
				}
			}
		]
	}`
	var prettyString bytes.Buffer
	ie := json.Indent(&prettyString, []byte(config), "", "  ")
	if ie != nil {
		log.Fatal(ie)
	}

	x := os.WriteFile(path, prettyString.Bytes(), 0644)
	if x != nil {
		log.Println("failed to create default config file at ", path)
		log.Fatal(x)
	}
}

func exist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		// just exit after logging
		log.Fatal(err)
		return false
	}
}
