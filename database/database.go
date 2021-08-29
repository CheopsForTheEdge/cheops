package database
//
//import (
//	"fmt"
//	"log"
//	"os/exec"
//)
//
//func LaunchDatabase() {
//    out, err := exec.Command("/bin/sh", "launch_db.sh").Output()
//    if err != nil {
//        log.Fatal(err)
//    }
//    fmt.Println(out)
//}

import (
	"context"
	"fmt"
	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"log"
	"os/exec"
)

func LaunchDatabase() {
    out, err := exec.Command("/bin/sh", "launch_db.sh").Output()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(out)
}
func Connection() driver.Client {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		// Handle error
		log.Fatal(err)
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		// Handle error
		log.Fatal(err)
	}
	return c
}

func ConnectToDatabase(client driver.Client) driver.Database {
	ctx := context.Background()
	db, err := client.Database(ctx, "myDB")
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	return db
}

func CollectionExists(db driver.Database, colName string) bool {
	ctx := context.Background()
	found, err := db.CollectionExists(ctx, colName)
	if err != nil {
		// handle error
	}
	return found
}

func CreateCollection(db driver.Database, colName string) driver.Collection {
	if CollectionExists(db, colName ) {
		ctx := context.Background()
		options := &driver.CreateCollectionOptions{}
		col, err := db.CreateCollection(ctx, colName, options)
		if err != nil {
			// handle error
		}
		return col
	}
	return nil
}
