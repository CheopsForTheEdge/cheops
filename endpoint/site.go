package endpoint

type Site struct {
	SiteName		string  `json:"SiteName"`
	Address 		string  `json:"Address"`
}


// Collection name variable
var colnamesite = "site"


func CreateSite(siteName string, address string) *Site {
	return &Site{SiteName: siteName, Address: address}
}

