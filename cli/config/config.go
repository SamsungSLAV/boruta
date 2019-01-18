// package config provides functions for handling cli config file
package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Output
	Servers map[string]Server
}

type Output struct {
	GlobalPretty bool
}

type Server struct {
	Name string
	Host string
	Port int
}

var configFileLocations []string

func init() {
	defaultLocations()
}

func defaultLocations() {
	configFileLocations = []string{
		"/etc/boruta/cli.yml",
		"/usr/share/config/boruta/cli.yml",
		"/usr/local/share/config/boruta/cli.yml",
		"$HOME/.config/boruta/cli.yml",
	}
}

func NewConfig() (Config, error) {
	viper.SetConfigType("yaml")

	defaultSrv := Server{Name: "main", Host: "localhost", Port: 8487}
	cfg := Config{
		Output: Output{
			GlobalPretty: true,
		},
		Servers: map[string]Server{
			defaultSrv.Name: defaultSrv,
		},
	}
	return cfg, nil
}

func NewConfigFile(path string) error {
	//config, err := NewConfig()
	//if err != nil {
	//handle
	//}
	//save config to file
	return nil
}

func ValidateConfigFile(path string) error {
	//check ips
	//check ports
	//check duplicates
	//check if default exists
	return nil
}

// Viper returns global Viper instance for flag binding etc outside of the config pkg.
func Viper() *viper.Viper {
	return viper.GetViper()
}

/*
configuration files locations and precedence

*/

func (c Config) DefaultServer() Server {
	return Server{}
}

func FindConfig() (path []string) {
	for range configFileLocations {
		//check them
	}
	return []string{""}
}
