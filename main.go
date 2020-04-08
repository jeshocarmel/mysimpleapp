package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/julienschmidt/httprouter"
)

var client *redis.Client

func main() {

	client = redis.NewClient(&redis.Options{
		Addr:     "db:6379",
		Password: "foobared", // no password set
		DB:       0,          // use default DB
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)

	// err = client.Set("visits", "0", 0).Err()
	// if err != nil {
	// 	panic(err)
	// }

	_, err = client.Get("visits").Result()
	if err != nil {
		client.Set("visits", "0", 0).Err()
	}

	router := setupRoutes()
	http.ListenAndServe(":80", router)
}

func setupRoutes() *httprouter.Router {
	router := httprouter.New()
	router.GET("/", ping)
	router.GET("/ping", ping)

	return router
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

	err := client.Set("visits", newVal, 0).Err()
	if err != nil {
		panic(err)
	}
}

func getVisitCount() string {
	val, err := client.Get("visits").Result()
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
