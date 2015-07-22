package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	repos, err := GetInstalledRepos()
	PanicOn(err)

	js, _ := json.MarshalIndent(&repos, "", "  ")
	fmt.Printf("%s\n", js)
}

func PanicOn(err error) {
	if err != nil {
		panic(err)
	}
}
