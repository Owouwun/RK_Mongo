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

//The data that be sent to fill of the html file
type Table struct {
	Titles []string
	Rows   [][]interface{}
}

func tablePage(w http.ResponseWriter, r *http.Request) {
	ctx, collection := getCollection("seagull", "request")
	var data Table
	getTable(&data, ctx, collection)

	tmpl, _ := template.ParseFiles("table.html")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}
func getTable(data *Table, ctx context.Context, collection *mongo.Collection) {
	data.Titles = getTitles(data, ctx, collection)
	data.Rows = getRows(data, ctx, collection)
}
func getTitles(data *Table, ctx context.Context, collection *mongo.Collection) []string {
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
func getRows(data *Table, ctx context.Context, collection *mongo.Collection) [][]interface{} {
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)
	var rows [][]interface{}

	for cur.Next(ctx) {
		var result bson.D
		err = cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}

		row := make([]interface{}, len(data.Titles))
		for j := 0; j < len(data.Titles); j++ {
			row[j] = result[j+1].Value
		}
		rows = append(rows, row)
	}
	return rows
}

type LoginData struct {
	Status string
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	status, _ := r.Cookie("Status")
	var data LoginData

	if status != nil {
		data.Status = status.Value
	} else {
		data.Status = strconv.Itoa(http.StatusAccepted)
	}

	deleteCookie(w, "Status")

	tmpl, _ := template.ParseFiles("login.html")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}
func deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Expires: time.Unix(0, 0)})
}
func auth(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:   "Status",
		Value:  strconv.Itoa(authCheck(r.FormValue("login"), r.FormValue("password"))), //Строковое значение статуса проверки аутентификации для отправленных формой логина и пароля
		MaxAge: 1000,
	}
	http.SetCookie(w, cookie)

	if cookie.Value == strconv.Itoa(http.StatusAccepted) {
		http.Redirect(w, r, "/table.html", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login.html", http.StatusSeeOther)
	}

	/*
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
		err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		if err!=nil {http.Error(w, "Wrong password!", http.StatusUnauthorized)}

		http.Redirect(w, r, "/table.html", http.StatusSeeOther)
	*/
}
func authCheck(login string, password string) int {
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
				return http.StatusAccepted
			} else {
				return http.StatusForbidden
			}
		}
	}
	return http.StatusUnauthorized
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	var data Table
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

	rowNumber, err := strconv.Atoi(r.FormValue("row"))
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i <= rowNumber; i++ { //Переносим указатель на нужную строку
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

	var updDoc bson.D //UpdatingDocument
	for i := 0; i < len(row); i++ {
		updDoc = append(updDoc, bson.E{Key: titles[i], Value: row[i]})
	}

	var setVal bson.D //SettingValues
	for i := 0; i < len(row); i++ {
		setVal = append(setVal, bson.E{Key: titles[i], Value: r.FormValue(titles[i])})
	}
	setVal = bson.D{{"$set", setVal}}

	_, err = collection.UpdateOne(ctx, updDoc, setVal)
	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/table.html", http.StatusSeeOther)
}
func addEmp(w http.ResponseWriter, r *http.Request) {
	ctx, collection := getCollection("seagull", "request")
	titles := getTitles(nil, ctx, collection)

	var Doc bson.D
	for i := 0; i < len(titles); i++ {
		Doc = append(Doc, bson.E{Key: titles[i], Value: r.FormValue(titles[i])})
	}

	_, err := collection.InsertOne(ctx, Doc)
	if err != nil {
		log.Fatal(err)
	}
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
