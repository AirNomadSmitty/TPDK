package utils

import (
	"encoding/json"
	"fmt"
)

func See(i interface{}) {
	s := JsonFormat(i)
	fmt.Println(s)
}

func JsonFormat(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
