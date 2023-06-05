package utils

import (
	"fmt"
	"github.com/spf13/viper"
)

type Configurations struct {
	Application 		ApplicationConfiguration
	Database    		DatabaseConfiguration
	KnownSites       	[]SiteConfiguration
	LocalServices 		[]EndpointConfiguration
	LocalSite			SiteConfiguration
}

type ApplicationConfiguration struct {
	Name			string
	HeartbeatPort	string
}

type DatabaseConfiguration struct {
	DBName 			string
	DBUser    		string
	DBPassword 		string
	Collections 	[]string
}

type SitesConfiguration struct {
	Sites 	 		[]SiteConfiguration
}

type SiteConfiguration struct {
	SiteName		string
	Address 		string
}

type EndpointsConfiguration struct {
	Site 			[]EndpointConfiguration
}

type EndpointConfiguration struct {
	Service 		string
	Address 		string
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
	viper.SetDefault("localsite", SiteConfiguration{SiteName: "Cheops1",
		Address: "127.0.0.1"})

	err := viper.Unmarshal(&conf)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v \n", err)
	}

	return conf
}