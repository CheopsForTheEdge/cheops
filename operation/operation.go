package operation

import "cheops/database"

type Operation struct {
	Operation  			string    	`json:"Operation"`
	Sites				[]string 	`json:"Sites"`
	Platform			string      `json:"Platform"`
	Service 			string    	`json:"Service"`
	Resource   			string  	`json:"Resource"`
	PlatformOperation	string		`json:"PlatformOperation"`
	ExtraArgs			[]string	`json:"ExtraArgs"`
	Request		        string    `json:"Request"`
}

// Collection name variable
var colname = "operation"

func CreateOperation(operation string,
	sites []string, platform string,
	service string, resource string,
	platformOperation string,
	extraArgs []string, request string) string {
	op := Operation{Operation: operation, Sites: sites,
		Platform: platform, Service: service, Resource: resource,
		PlatformOperation: platformOperation, ExtraArgs: extraArgs,
		Request: request}
	return database.CreateResource(colname, op)
	}



func SearchEndpoints(op Operation) []string {
	var addresses []string
	for _, site := range op.Sites{
		addresses = append(addresses, site)
	}
	return addresses
}