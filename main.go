package main

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"GO-Server/operationsPackage"
)

// Global variables
var (
	PORT = os.Getenv("PORT")

)

func init(){
	// Initialize the port
	if PORT == "" {
		PORT = "8080"
	}
}

func main(){
	// Initializing the router
	router := httprouter.New()

	router.GET("/", operationsPackage.Index)
	// GetDataFromMysql gets only the itemId which are available.
	router.GET("/getData",operationsPackage.GetDataFromMysql)  //opeartionsPackage.GetData is older one, which gets all the items with the properties of name, price etc.
	log.Print("Server is ready now...")

	log.Fatal(http.ListenAndServe(":"+PORT, router))


}
