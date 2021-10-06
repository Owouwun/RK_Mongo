package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"html/template"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getCollection(dbName string, collectionName string) (context.Context, *mongo.Collection) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// Create connect
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	collection := client.Database(dbName).Collection(collectionName)
	return ctx, collection
}

//The data that be sended to fill of the html file
type Data struct {
	Table Table
}
type Table struct {
	Rows   [][]interface{}
	Titles []string
}

func getTable(data *Data, ctx context.Context, collection *mongo.Collection) {
	getTitles(data, ctx, collection)
	getRows(data, ctx, collection)
}
func getTitles(data *Data, ctx context.Context, collection *mongo.Collection) {
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	cur.Next(ctx)
	var result bson.D
	err = cur.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	titles := make([]string, len(result)-1)
	for i := 0; i < len(titles); i++ {
		titles[i] = result[i+1].Key
	}

	data.Table.Titles = titles
}
func getRows(data *Data, ctx context.Context, collection *mongo.Collection) {
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	var rows [][]interface{}

	for i := 0; cur.Next(ctx); i++ {
		var result bson.D
		row := make([]interface{}, len(data.Table.Titles))

		err = cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}
		for j := 0; j < len(data.Table.Titles); j++ {
			row[j] = result[j+1].Value
		}
		rows = append(rows, row)
	}
	data.Table.Rows = rows
}
func table(w http.ResponseWriter, r *http.Request) {
	var data Data
	ctx, collection := getCollection("seagull", "request")
	getTable(&data, ctx, collection)
	tmpl, _ := template.ParseFiles("table.html")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func authCheck(login string, password string) bool {
	ctx, collection := getCollection("seagull", "employee")
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	for cur.Next(ctx) {
		var result bson.D

		err = cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}
		m := result.Map()
		if m["login"] == login {
			if m["password"] == password {
				println("Верный пароль.")
				return true
			} else {
				println("Неверный пароль!")
				return false
			}
		}
	}
	println("Такого логина не существует!")
	return false
}

func auth(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	println(login, " ", password)

	if authCheck(login, password) {
		http.Redirect(w, r, "/table.html", http.StatusSeeOther)
	}

	/*
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
		err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		if err!=nil {http.Error(w, "Wrong password!", http.StatusUnauthorized)}

		http.Redirect(w, r, "/table.html", http.StatusSeeOther)
	*/
}

func main() {
	http.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := template.ParseFiles("login.html")
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Fatal(err)
		}
	})
	http.HandleFunc("/auth", auth)

	http.HandleFunc("/table.html", table)

	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server is listening...")
}
