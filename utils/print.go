package utils

import (
	"encoding/json"
	"fmt"
)

// PrittyPrint prints the given interface in a pretty format
func PrittyPrint(v ...interface{}) {
	for _, i := range v {
		b, err := json.MarshalIndent(i, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	}
}
