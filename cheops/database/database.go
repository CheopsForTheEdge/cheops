package database



import (
)

type Database struct {
	user      string  `json:"user"`
	password  string  `json:"password"`
	address   string  `json:"address"`
	port      string  `json:"port"`
	protocol  string  `json:"protocol"`
	endpoint  string  `json:"endpoint"`
}


func NewDatabase(user string,
				password string,
				address string,
				port string,
				protocol string,
				endpoint string) *Database {
	return &Database{user: user,
					password: password,
					address: address,
					port: port,
					protocol: protocol,
					endpoint: endpoint}
}


// Connect to the database


// Add an entry to the database


// Edit an entry in the database


// Delete an entry in the database