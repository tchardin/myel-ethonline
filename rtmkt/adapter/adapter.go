package main

import "C"
import (
	"fmt"

	rtmkt "github.com/tchardin/myel-ethonline/rtmkt/lib"
)

var node *rtmkt.MyelNode

//export StartNode
func StartNode() *C.char {
	var err error
	node, err = rtmkt.SpawnNode(rtmkt.NodeTypeProvider)
	if err != nil {
		return C.CString(fmt.Sprintf("Unable to start node: %v", err))
	}
	return nil
}

func main() {}
