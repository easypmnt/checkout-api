package utils

import (
	"encoding/json"
	"fmt"
)

// PrettyPrint prints the given interface in a pretty format
func PrettyPrint(v ...interface{}) {
	for _, i := range v {
		b, err := json.MarshalIndent(i, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	}
}
