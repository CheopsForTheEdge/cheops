package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Configurations struct {
	Application		ApplicationConfigurations
	Database 		DatabaseConfigurations
}

type ApplicationConfigurations struct {
	Name		string
}

type DatabaseConfigurations struct {
	DBName 		string
	DBUser    	string
	DBPassword 	string
	Collections []string
}

var Conf = LoadConfig()

func LoadConfig() Configurations {
	// From https://medium.com/@bnprashanth256/reading-configuration-files-and-environment-variables-in-go-golang-c2607f912b63
	// Set the file name of the configurations file
	viper.SetConfigName("config")

	// Set the path to look for the configurations file
	viper.AddConfigPath(".")

	// Enable VIPER to read Environment Variables
	viper.AutomaticEnv()

	viper.SetConfigType("yml")
	var conf Configurations

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	// Set undefined variables
	viper.SetDefault("database.dbname", "cheops")
	viper.SetDefault("database.dbuser", "cheops@cheops")
	viper.SetDefault("database.collections", []string{"operations",
		"replications",
		"endpoints",
		"sites",
		"crosses"})

	return conf
}