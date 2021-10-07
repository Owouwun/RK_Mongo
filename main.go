package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"html/template"
	"log"
	"net/http"
	"strconv"
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
	Error bool
}
type Table struct {
	Titles []string
	Rows   [][]interface{}
}

func getTable(data *Data, ctx context.Context, collection *mongo.Collection) {
	data.Table.Titles = getTitles(data, ctx, collection)
	data.Table.Rows = getRows(data, ctx, collection)
}
func getTitles(data *Data, ctx context.Context, collection *mongo.Collection) []string {
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

	return titles
}
func getRows(data *Data, ctx context.Context, collection *mongo.Collection) [][]interface{} {
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
	return rows
}
func tablePage(w http.ResponseWriter, r *http.Request) {
	var data Data
	ctx, collection := getCollection("seagull", "request")
	getTable(&data, ctx, collection)
	tmpl, _ := template.ParseFiles("table.html")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("Status")
	var unauth []bool

	//data.Status = append(data.Status, "")

	if c != nil && c.Value == "401" {
		c = &http.Cookie{
			Name:    "Status",
			Expires: time.Unix(0, 0),
		}
		http.SetCookie(w, c)
		unauth = append(unauth, true)
		//data.Status = append(data.Status, "Неверный логин или пароль!")
	} else {
		unauth = append(unauth, false)
	}
	tmpl, _ := template.ParseFiles("login.html")
	err := tmpl.Execute(w, unauth)
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
	} else {
		cookie := &http.Cookie{
			Name:   "Status",
			Value:  "401",
			MaxAge: 1000,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/login.html", http.StatusSeeOther)
	}

	/*
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
		err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		if err!=nil {http.Error(w, "Wrong password!", http.StatusUnauthorized)}

		http.Redirect(w, r, "/table.html", http.StatusSeeOther)
	*/
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	var data Data
	ctx, collection := getCollection("seagull", "employee")
	getTable(&data, ctx, collection)
	tmpl, _ := template.ParseFiles("admin.html")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}
func updateEmp(w http.ResponseWriter, r *http.Request) {
	ctx, collection := getCollection("seagull", "request")

	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	n, err := strconv.Atoi(r.FormValue("row"))
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i <= n; i++ {
		cur.Next(ctx)
	}
	var result bson.D
	err = cur.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	row := make([]interface{}, len(result)-1)

	for i := 0; i < len(result)-1; i++ {
		row[i] = result[i+1].Value
	}

	titles := getTitles(nil, ctx, collection)

	var EsDoc []bson.E
	for i := 0; i < len(row); i++ {
		EsDoc = append(EsDoc, bson.E{Key: titles[i], Value: row[i]})
	}
	var updDoc bson.D
	updDoc = EsDoc

	var EsVal []bson.E
	for i := 0; i < len(row); i++ {
		EsVal = append(EsVal, bson.E{Key: titles[i], Value: r.FormValue(titles[i])})
	}
	var D bson.D
	D = EsVal
	setVal := bson.D{{"$set", D}}
	res, err := collection.UpdateOne(ctx, updDoc, setVal)
	if err != nil {
		log.Fatal(err)
	}
	println(res.MatchedCount, res.ModifiedCount)
	http.Redirect(w, r, "/table.html", http.StatusSeeOther)
}
func addEmp(w http.ResponseWriter, r *http.Request) {
	ctx, collection := getCollection("seagull", "request")

	titles := getTitles(nil, ctx, collection)

	var EsVal []bson.E
	for i := 0; i < len(titles); i++ {
		EsVal = append(EsVal, bson.E{Key: titles[i], Value: r.FormValue(titles[i])})
	}
	var Doc bson.D
	Doc = EsVal
	//setVal := bson.D{{"$set", D}}
	res, err := collection.InsertOne(ctx, Doc)
	if err != nil {
		log.Fatal(err)
	}
	println(res.InsertedID)
	http.Redirect(w, r, "/table.html", http.StatusSeeOther)
}

func main() {
	http.HandleFunc("/login.html", loginPage)
	http.HandleFunc("/auth", auth)

	http.HandleFunc("/table.html", tablePage)
	http.HandleFunc("/updateEmp", updateEmp)
	http.HandleFunc("/addEmp", addEmp)

	http.HandleFunc("/admin.html", adminPage)

	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server is listening...")
}
