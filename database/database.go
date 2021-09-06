package database

import (
	"context"
	"fmt"
	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"log"
	"os/exec"
	"time"
)


func LaunchDatabase() {
	out, err := exec.Command("/bin/sh", "database/launch_db.sh").Output()
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
	conn.SetAuthentication(driver.BasicAuthentication("root", ""))
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		// Handle error
		log.Fatal(err)
	}
	return c
}


func CreateDatabase(client driver.Client, dbname string) driver.Database {
	ctx := context.Background()
	exists, err := client.DatabaseExists(ctx, dbname)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if !exists {
		dbdefault := driver.CreateDatabaseDefaultOptions{}
		user:= driver.CreateDatabaseUserOptions{UserName:"cheops",
			Password:"cheops"}
		options := &driver.CreateDatabaseOptions{Users:[]driver.CreateDatabaseUserOptions{user},
			Options: dbdefault}
		db, err := client.CreateDatabase(ctx, dbname, options)
		if err != nil {
			// handle error
			log.Fatal(err)
		}
		return db
	}
	return nil
}


func ConnectToDatabase(client driver.Client, dbname string) driver.Database {
	ctx := context.Background()
	exists, err := client.DatabaseExists(ctx, dbname)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if exists {
		db, err := client.Database(ctx, dbname)
		if err != nil {
			// handle error
			log.Fatal(err)
		}
		return db
	}
	return nil
}


func ConnectToCollection(db driver.Database, colName string) driver.Collection  {
	ctx := context.Background()
	exists, err := db.CollectionExists(ctx, colName)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if exists {
		col, err := db.Collection(ctx, colName)
		if err != nil {
			// handle error
			log.Fatal(err)
		}
		return col
	}
	return nil
}


func CreateCollection(db driver.Database, colName string) driver.Collection {
	ctx := context.Background()
	exists, err := db.CollectionExists(ctx, colName)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if !(exists) {
		options := &driver.CreateCollectionOptions{}
		col, err := db.CreateCollection(ctx, colName, options)
		if err != nil {
			// handle error
			log.Fatal(err)
		}
		return col
	}
	return nil
}


func PrepareForExecution(dbname string, colname string) (driver.Database, driver.Collection) {
	LaunchDatabase()
	time.Sleep(15 * time.Second)
	c := Connection()
	CreateDatabase(c, dbname)
	db := ConnectToDatabase(c, dbname)
	col := CreateCollection(db, colname)
	ExecuteQuery(db)
	return db, col
}



func ExecuteQuery(db driver.Database) bool {
	// ctx := context.Background()
	fmt.Println("test")
	return true
}

func CreateResource(col driver.Collection, doc interface{}) {
	ctx := context.Background()
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		// handle error
	}
	fmt.Printf("Created document with key '%s', revision '%s'\n", meta.Key, meta.Rev)
}

func ReadResource() {}

func UpdateResource() {}

func DeleteResource() {}