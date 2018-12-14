package operationsPackage

import (
	"cloud.google.com/go/datastore"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)
var db *sql.DB

var maxResultsOnce int
var user_id string
// To know we have user data in our model, else there is no need of calling getRecFromModel() when user scrolls the page
var USER_MODEL_PRESENT = true
var USE_LOCALHOST_DB = false
// Context declaration - Global variable for Datastore

var ctx = context.Background()



var (
	mysqlUser = os.Getenv("MYSQL_USER")
	mysqlPass = os.Getenv("MYSQL_PASS")
	mysqlDB   = os.Getenv("MYSQL_DB")
	mysqlHost = os.Getenv("MYSQL_HOST")
	mysqlPort = os.Getenv("MYSQL_PORT")
)

type Item struct {
	Id          string    `json:"id"`
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Item_condition int  `json:"item_condition"`
	Num_likes int  `json:"num_likes"`

}

type Model struct{
	AvgPrice []int
	Category []int
	ItemCondition []string
	StdDev []int

}

type Trending struct{
	Category string

}
type Entity struct {
	//Description string
	Recommend []Recommends
}

type Recommends struct {
	Name string
	Price int
	Index int
	Num_likes int
	Item_id string
}


func init(){
	// Number of results to be shown to user on request, if they scroll down, next we show number of 'maxResultsOnce' items
	maxResultsOnce = 10
	// Use db = connectMySQL() when using localhost only
	if (USE_LOCALHOST_DB == true) {
		db = connectMySQL()
	}


}



func connectMySQL() *sql.DB {
	// Set defaults
	if mysqlUser == "" {
		mysqlUser = "root"
	}
	if mysqlDB == "" {
		mysqlDB = "recommend"
	}
	if mysqlPass==""{
		mysqlPass = "pass"
	}
	if mysqlHost == "" {
		mysqlHost = "127.0.0.1"
	}
	if mysqlPort == "" {
		mysqlPort = "3306"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", mysqlUser, mysqlPass, mysqlHost, mysqlPort, mysqlDB)
	fmt.Println("Connecting mysql at: %s", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err.Error())
	}
	if err = db.Ping(); err != nil {
		panic(err.Error())
	}
	return db
}

func Index(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("method:", r.Method) //get request method
	if r.Method == "GET" {

		t, _ := template.ParseFiles("front.html")
		t.Execute(w, nil)
	} else {

		r.ParseForm() // Parse the form
		fmt.Println("userid:", r.Form["userid"])

	}
}

// Executes the query
func executeMyQuery(query string)[]Item{

	const instanceConnection= "kouzoh-p-codechaitu:asia-east1:mysqldb"
	var cfg= mysql.Cfg(instanceConnection, "root", "pass")
	cfg.DBName = "recommend"
	db, err:= mysql.DialCfg(cfg)
	if err != nil{
		fmt.Print(err)
	}
	result, err := db.Query(query)
	if err != nil{
		log.Printf("Failed to select from db: %s", err)

	}

	outputItems := make([]Item, 0)

	for result.Next(){
		var item  Item
		err = result.Scan(&item.Id, &item.Name, &item.Price, &item.Item_condition, &item.Num_likes)

		if err !=nil{
			log.Printf("Failed to scan row: %s", err)
		}
		//fmt.Println("From DB: "+ out)
		outputItems = append(outputItems, item)
	}
	fmt.Println(outputItems)


	return outputItems
}

// This function will return the selected items from user previous purchases
func recFromUserModel(user_id string, offset int) []Item{
	var modelObj  Model
	var DsClient, err = datastore.NewClient(ctx, "kouzoh-p-codechaitu")
	if err != nil {
		// Handle error.
		fmt.Println("error in creating client")
	}
	key := datastore.NameKey("Model", user_id, nil)
	if err := DsClient.Get(ctx, key, &modelObj); err != nil {
		// Handle error.
		fmt.Println("error in getting data"+fmt.Sprint(err))

		// Return nothing,if the key given is not present
		// Make the flag such that user's model is not present
		USER_MODEL_PRESENT = false
		return make([]Item,0)
	}

	// Considering only top first category for getting items, so using [0] items from modelObj

	cat, price, stddev, ic := modelObj.Category[0], modelObj.AvgPrice[0], modelObj.StdDev[0], modelObj.ItemCondition[0]
	if( stddev==0){
		// If standard dev is 0, then to select items with half less than price range from mean.
		stddev = price / 2
	}
	query := "SELECT id, name, price, item_condition, num_likes FROM recommend.ITEMS_DETAILS where category_id="+strconv.Itoa(cat)+" and price between "+strconv.Itoa(int(math.Abs(float64(price)-float64(stddev)))) +" and "+(strconv.Itoa(price+stddev))+" and item_condition="+ic+ " order by  num_likes desc "+" LIMIT "+ strconv.Itoa(offset)+","+strconv.Itoa(maxResultsOnce)
	fmt.Println("In Model")
	return executeMyQuery(query)

}

// recFromTrending will return the items from trending category
func recFromTrending(offset int)[]Item{
	var trendObj Trending
	var DsClient, err = datastore.NewClient(ctx, "kouzoh-p-codechaitu")
	if err != nil {
		// Handle error.
		fmt.Println("error in creating client")
	}
	key := datastore.NameKey("Trending", "trend", nil)
	if err := DsClient.Get(ctx, key, &trendObj); err != nil {
		// Handle error.
		fmt.Println("error in getting data"+fmt.Sprint(err))
	}

	// Take top category from which users purchased items last day
	cat := trendObj.Category
	fmt.Println(cat)
	query := "SELECT id, name, price, item_condition, num_likes FROM recommend.ITEMS_DETAILS where category_id in("+cat+") order by num_likes desc LIMIT "+ strconv.Itoa(offset)+","+strconv.Itoa(maxResultsOnce)
	fmt.Println("In trending "+query)
	return executeMyQuery(query)

}
func GetDataFromMysql(w http.ResponseWriter, r *http.Request, ps httprouter.Params){

	r.ParseForm()

	var user_id = strings.Join(r.Form["userid"], "")
	pageCounter := strings.Join(r.Form["page"], "")

	// Fetch the results from the given offset.
	offset,_ := strconv.Atoi(pageCounter)
	offset = offset * maxResultsOnce

	fmt.Println("userId is :", user_id)

	// We generate recommendations based on 1) previous user purchases and 2) trending items
	// 1) From model generated, recommendations are as,
	fmt.Println("-----Gonna print-----")
	var modelRecommendation []Item
	if USER_MODEL_PRESENT == true {
		modelRecommendation = recFromUserModel(user_id, offset)
		fmt.Println(modelRecommendation)
	}
	// 2) From trending item categories, get the recommendations
	var trendingRecommendation = recFromTrending(offset)
	fmt.Println(trendingRecommendation)
	/// After trendingRecommendation, ... dots indicate to append two lists
	recommendations := append(modelRecommendation, trendingRecommendation...)

	fmt.Println("before sending")
	json.NewEncoder(w).Encode(recommendations)

}




/* GetData function is going to be depriciated */

func GetData(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//startTime := time.Now()
	r.ParseForm()
	user_id= strings.Join(r.Form["userid"], "")
	fmt.Println("userNum is :", user_id)
	ctx := context.Background()
	dsClient, err := datastore.NewClient(ctx, "kouzoh-p-codechaitu")
	if err != nil {
		// Handle error.
		fmt.Println("error in creating client")
	}
	user_id_int64, _ := strconv.ParseInt(user_id, 10, 64)
	k := datastore.IDKey("Recommendations", user_id_int64, nil)

	var e Entity
	if err := dsClient.Get(ctx, k, &e); err != nil {
		// Handle error.
		fmt.Println("error in getting data" + fmt.Sprint(err))
	}
	Len := len(e.Recommend)
	output := "["
	for i := 0; i < Len; i++ {
		m := Recommends{e.Recommend[i].Name, e.Recommend[i].Price, e.Recommend[i].Index, e.Recommend[i].Num_likes, e.Recommend[i].Item_id}
		data, err2 := json.Marshal(m)
		if err2 != nil {
			fmt.Println("wrong in json " + fmt.Sprint(err2))
		}
		if (i < Len-1) {

			output = output + string(data) + ","
		} else {

			output = output + string(data)
		}

	}
	output = output + "]"
	fmt.Println(output)
	fmt.Fprint(w, output)

}