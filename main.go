package main

import "log"

func main() {
	app := NewApp()
	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}