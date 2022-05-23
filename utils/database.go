package utils

import (
	"context"
	"fmt"
	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"log"
	"os/exec"
	"time"
)

var dbcheops = "cheops"
var database = Conf.Database

func LaunchDatabase() {
	_, err := exec.Command("/bin/sh", "utils/launch_db.sh").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database ready")
}


func Connection() driver.Client {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		// Handle error
		log.Fatal(err)
	}
	conn.SetAuthentication(driver.BasicAuthentication(database.DBUser, database.DBPassword))
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		// Handle error
		log.Fatal(err)
	}
	return c
}


func CreateDatabase(client driver.Client) driver.Database {
	ctx := context.Background()
	exists, err := client.DatabaseExists(ctx, dbcheops)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if !exists {
		dbdefault := driver.CreateDatabaseDefaultOptions{}
		user:= driver.CreateDatabaseUserOptions{UserName: database.DBUser,
			Password: database.DBPassword}
		options := &driver.CreateDatabaseOptions{Users:[]driver.CreateDatabaseUserOptions{user},
			Options: dbdefault}
		db, err := client.CreateDatabase(ctx, dbcheops, options)
		if err != nil {
			// handle error
			log.Fatal(err)
		}
		return db
	}
	return nil
}


func ConnectToDatabase(client driver.Client) driver.Database {
	ctx := context.Background()
	exists, err := client.DatabaseExists(ctx, dbcheops)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	if exists {
		db, err := client.Database(ctx, dbcheops)
		if err != nil {
			fmt.Println("Can't connect to database")
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
		fmt.Println("Can't check if collection exists")
		log.Fatal(err)
	}
	if exists {
		col, err := db.Collection(ctx, colName)
		if err != nil {
			// handle error
			fmt.Println("Can't connect to collection")
			log.Fatal(err)
		}
		return col
	} else {
		fmt.Println("Collection does not exists")
		log.Fatal(err)
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
			fmt.Println("Can't create collection")
			log.Fatal(err)
		}
		return col
	} else {
		fmt.Println("Collection already exists")
	}

	return nil
}

func CreateCollectionWithIndexes(db driver.Database, colName string, fields []string) driver.Collection {
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
			fmt.Println("Can't create collection")
			log.Fatal(err)
		}
		_, _, error := col.EnsurePersistentIndex(ctx, fields, nil)
		if error != nil {
			// handle error
			fmt.Println("Can't create index")
			log.Fatal(err)
		}
		return col
	} else {
		fmt.Println("Collection already exists")
	}

	return nil
}


func PrepareForExecution() (driver.Database,
	[]driver.Collection) {
	LaunchDatabase()
	time.Sleep(15 * time.Second)
	c := Connection()
	CreateDatabase(c)
	db := ConnectToDatabase(c)
	var cols []driver.Collection
	for _, col := range database.Collections {
		c := CreateCollection(db, col)
		cols = append(cols, c)
	}
	return db, cols
}


func ConnectionToCheopsDatabase() (driver.Database){
	c := Connection()
	db := ConnectToDatabase(c)
	return db
}

func ConnectionToCorrectCollection(colname string) (driver.Collection){
	c := Connection()
	db := ConnectToDatabase(c)
	col := ConnectToCollection(db, colname)
	return col
}

 func ExecuteQuery(query string, bindVars map[string]interface{},
 result interface{}) (
 	cursor driver.Cursor) {
 	ctx := context.Background()
 	db := ConnectionToCheopsDatabase()
 	cursor, err := db.Query(ctx, query, bindVars)
 	if err != nil {
		 fmt.Println("Can't execute the query")
		 log.Fatal(err)
		 // handle error
 	}
	cursor.ReadDocument(ctx, &result)
 	return cursor
}

func CreateResource(colname string, doc interface{}) string {
	ctx := context.Background()
	col := ConnectionToCorrectCollection(colname)
	meta, err := col.CreateDocument(ctx, doc)
	if err != nil {
		fmt.Println("Can't create the resource")
		log.Fatal(err)
		// handle error
	}
	return meta.Key
}


func ReadResource(colname string, key string, doc interface{}) {
	ctx := context.Background()
	col := ConnectionToCorrectCollection(colname)
	_, err := col.ReadDocument(ctx, key, doc)
	if err != nil {
		fmt.Println("Can't access the resource")
		log.Fatal(err)
	}
}


// Key is ArangoDB key, patch is the changed part
func UpdateResource(colname string, key string, patch interface{}) {
	ctx := context.Background()
	col := ConnectionToCorrectCollection(colname)
	_, err := col.UpdateDocument(ctx, key, patch)
	if err != nil {
		fmt.Println("Can't access the resource")
		log.Fatal(err)
	}
}

func DeleteResource(colname string, key string) {
	ctx := context.Background()
	col := ConnectionToCorrectCollection(colname)
	_, err := col.RemoveDocument(ctx, key)
	if err != nil {
		fmt.Println("Can't remove the resource")
		log.Fatal(err)
	}
}

func DeleteResourceFromSearch(colname string, key string, value string) {
	var result interface{}
	SearchResource(colname, key, value, result)
	// TODO: how to get the key?
	// DeleteResource(colname, result.Key)
}

//func GetAllResources(colname string) ([]interface{}){
//	var lst []interface{}
//	ctx := context.Background()
//	query := "FOR d IN @colname RETURN d"
//	db := ConnectionToCheopsDatabase()
//	cursor, err := db.Query(ctx, query, nil)
//	db.
//	if err != nil {
//		fmt.Println("A problem arrived when getting the resources of %s:\n",
//			colname)
//		log.Fatal(err)
//	}
//	defer cursor.Close()
//	return lst
//}


func SearchResource(colname string, key string,
	value string, result interface{}) (interface{}, string) {
	ctx := context.Background()
	db := ConnectionToCheopsDatabase()
	query := "FOR doc IN @colname\n" +
		"SEARCH @key == @value\n" +
		"RETURN doc"
	bindvars := map[string]interface{} { "@colname": colname, "@key": key,
		"@value": value}
	cursor, err := db.Query(ctx, query,	bindvars)
		if err != nil {
			fmt.Println("Can't execute the query")
			log.Fatal(err)
			// handle error
		}
	docmeta, _  := cursor.ReadDocument(ctx, &result)
	fmt.Println(result)
	defer cursor.Close()
	return &result, docmeta.Key
}
