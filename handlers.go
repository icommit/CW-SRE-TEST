// Main package for ClouldWalk SRE-Test
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/icommit/SRETest/core"
)

// Main hanlder. The frontend project has one endpoint; "/"
// Pretty simple and straightforward; we parse our html file and pass in our
// logwarehouse data. Logic to opt-in to email notification is defined here.
func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ctx := context.Background()
	client := core.CreateClient(ctx)
	store := client.Collection("config").Doc("config")

	ts, err := template.ParseFiles("./ui/html/home.html")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Println(w, http.StatusBadRequest)
			return
		}
		email := r.FormValue("email")
		toggle := r.FormValue("toggle")
		//toggle, _ := strconv.ParseBool(r.FormValue("toggle"))

		if email != "" {
			_, err := store.Set(ctx, map[string]interface{}{
				"email": email,
			}, firestore.MergeAll)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("Set: An error has occurred: %s", err)
			}
		}
		if toggle == "on" {
			_, err := store.Set(ctx, map[string]interface{}{
				"update": true,
			}, firestore.MergeAll)
			if err != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("Set: An error has occurred: %s", err)
			}
		} else {
			_, errs := store.Set(ctx, map[string]interface{}{
				"update": false,
			}, firestore.MergeAll)
			if errs != nil {
				// Handle any errors in an appropriate way, such as returning them.
				log.Printf("Set: An error has occurred: %s", errs)
			}
		}
	}

	err = ts.Execute(w, warehouse)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}
