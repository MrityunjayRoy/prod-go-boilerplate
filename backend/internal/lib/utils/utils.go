package utils

import (
	"encoding/json"
	"fmt"
)

func PrintJSON(v interface{}) {
	json, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		fmt.Println("Error marshalling the json:", err)
	}

	fmt.Println("JSON:", string(json))
}
