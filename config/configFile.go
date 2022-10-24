package config

import "os"
import "gopkg.in/yaml.v3"

type ConfigFile struct {
	filePath string
	Config Config
}

func (configFile *ConfigFile) Load() error {
	f, err := os.Open(configFile.filePath)
	if err != nil { return err }
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&configFile.Config); err != nil { return err }
	return nil
}

func (configFile *ConfigFile) Save() error {
	f, err := os.OpenFile(configFile.filePath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil { return err }
	defer f.Close()
	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(configFile.Config); err != nil { return err }
	return nil
}



func NewConfigFile(filePath string) (*ConfigFile, error) {
	configFile := ConfigFile{
		filePath: filePath,
	}
	if err := configFile.Load(); err != nil { return nil, err }
	return &configFile, nil
}