package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-redis/redis/v7"
	"github.com/julienschmidt/httprouter"
)

var cacheClient *redis.Client
var db *sql.DB
var templates *template.Template
var err error

func init() {

	connectRedisCache()
	connectDB()

	// funcmap for go templates
	funcMap := template.FuncMap{

		"dateToString": func(t1 time.Time) string {
			return t1.Format("2-Jan-2006")
		},

		"timeToString": func(t1 time.Time) string {
			return t1.Format("January 02, 2006 15:04")
		},
	}

	templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.gohtml"))
}

func connectDB() {
	//initiate DB
	dbDriver := "mysql"
	dbHost := "db"
	dbUser := "root"
	dbPass := os.Getenv("MYSQL_ROOT_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")

	db, err = sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp("+dbHost+")/"+dbName+"?parseTime=true")

	if err != nil {
		log.Println("Error: ", err.Error())
	} else {
		log.Println("DB connection successful !!", db.Ping())
	}
}

func connectRedisCache() {
	cacheClient = redis.NewClient(&redis.Options{
		Addr:     "cache:6379",
		Password: os.Getenv("REDIS_PASSWORD"), // no password set
		DB:       0,                           // use default DB
	})

	pong, err := cacheClient.Ping().Result()
	log.Println(pong, err)

	_, err = cacheClient.Get("visits").Result()
	if err != nil {
		cacheClient.Set("visits", "0", 0).Err()
	}
}

func main() {

	router := setupRoutes()
	http.ListenAndServe(":80", router)
}

func setupRoutes() *httprouter.Router {
	router := httprouter.New()
	router.GET("/", home)
	router.GET("/ping", ping)
	router.GET("/simpleform", simpleForm)
	router.POST("/simpleformpost", simpleFormPost)
	router.GET("/simple_form_last5", simpleFormLast5)

	router.ServeFiles("/static/*filepath", http.Dir("static"))

	return router
}

func home(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	// to catch panics and return
	defer throwErrors(w)

	incrementVisitCount()
	templates.ExecuteTemplate(w, "index.gohtml", getVisitCount())

}

func simpleForm(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	defer throwErrors(w)
	templates.ExecuteTemplate(w, "simple_form.gohtml", nil)
}

func simpleFormPost(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	defer throwErrors(w)
	r.ParseForm()

	//insert to db
	stmt, err := db.Prepare("insert into SIMPLE_FORM_DATA(first_name, last_name, country, favourite_food) VALUES (?,?,?,?)")

	res, err := stmt.Exec(r.FormValue("first_name"), r.FormValue("last_name"), r.FormValue("country"), r.FormValue("favourite_food"))
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	//get the inserted kit_id
	id, _ := res.LastInsertId()
	fmt.Println("inserted id: ", id)

	templates.ExecuteTemplate(w, "simple_form_success.gohtml", getLast5SubmittedFormRecords())
}

//Records ...
type Records struct {
	FirstName     string
	LastName      string
	Country       string
	FavouriteFood string
	AddedDate     time.Time
}

func simpleFormLast5(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	defer throwErrors(w)
	templates.ExecuteTemplate(w, "simple_form_last5.gohtml", getLast5SubmittedFormRecords())

}

func getLast5SubmittedFormRecords() []Records {

	rows, err := db.Query("SELECT first_name, last_name, country, favourite_food, added_date FROM SIMPLE_FORM_DATA order by added_date desc limit 5")
	if err != nil {
		log.Println(err.Error())
		panic(err.Error())
	}

	last5records := []Records{}

	for rows.Next() {
		var r Records
		err = rows.Scan(&r.FirstName, &r.LastName, &r.Country, &r.FavouriteFood, &r.AddedDate)
		if err != nil {
			log.Println(err.Error())
			panic(err.Error())
		}
		last5records = append(last5records, r)
	}

	return last5records

}

func ping(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	// to catch panics and return
	defer throwErrors(w)

	incrementVisitCount()

	response := struct {
		Message string
		Visits  string
	}{
		Message: "This is a simple ping message from the App",
		Visits:  getVisitCount(),
	}

	byteArr, _ := json.Marshal(&response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(byteArr)
}

func incrementVisitCount() {

	curr := getVisitCount()
	currValInt, _ := strconv.Atoi(curr)
	newVal := strconv.Itoa(currValInt + 1)

	err := cacheClient.Set("visits", newVal, 0).Err()
	if err != nil {
		panic(err)
	}
}

func getVisitCount() string {
	val, err := cacheClient.Get("visits").Result()
	if err != nil {
		panic(err)
	}
	return val
}

func throwErrors(w http.ResponseWriter) { //catch or finally
	if err := recover(); err != nil { //catch
		msg := fmt.Errorf("%v", err).Error()
		log.Println(err)

		response := struct {
			Code  int
			Error string
		}{
			Code:  400,
			Error: msg,
		}

		byteArr, _ := json.Marshal(&response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(byteArr)
		w.WriteHeader(http.StatusBadRequest)
	}
}
