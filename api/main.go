package main

// #include <stdio.h>
// #include <stdlib.h>
//
// static void myprint(char* s) {
//   printf("%s\n", s);
// }
import "C"
import (
	"fmt"
	"os"
	"os/signal"
	"unsafe"
)

func main() {
	err := SpawnIpfsNode()

	if err != nil {
		C.myprint(err)
	}

	nid := GetNodeId()

	C.free(unsafe.Pointer(nid))

	testCid := C.CString("QmZ4e4RgtU2oJsDZAu4RyZkeXnRnEb5u78S9EG1DeKvmto")
	err = GetFile(testCid)
	if err != nil {
		C.myprint(err)
	}

	testUrl := C.CString("https://images.unsplash.com/photo-1601666703585-964591b026c5")
	resCid, err := AddWebFile(testUrl)

	if err != nil {
		C.myprint(err)
	}

	fmt.Printf("Result cid: %v\r\n", C.GoString(resCid))

	err = SpawnFilecoinNode()
	if err != nil {
		fmt.Printf("Unable to spawn Filecoin node: %v\r\n", C.GoString(err))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	select {
	case <-stop:
		fmt.Println("Shutting down")
		CloseNode()
		os.Exit(0)
	}
}
