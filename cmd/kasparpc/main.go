package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
)

var kaspad *rpcclient.RPCClient

type hRpcError struct {
	ErrorMsg string `json:"error"`
}

func rpcRetAny(w http.ResponseWriter, r *http.Request, rObj any) {
	jMsg, err := json.Marshal(rObj)

	if err != nil {
		log.Printf("Error Marshal jMsg %s", err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(jMsg)
}

func rpcRetAll(w http.ResponseWriter, r *http.Request) {
	var aMsg any
	var err error
	switch r.URL.Path {
	case "/GetBlocks":
		lBlock := r.URL.Query().Get("lowHash")
		aMsg, err = kaspad.GetBlocks(lBlock, true, true)
	case "/GetVirtualSelectedParentChainFromBlock":
		lBlock := r.URL.Query().Get("startHash")
		aMsg, err = kaspad.GetVirtualSelectedParentChainFromBlock(lBlock, true)
	case "/GetUtxosByAddresses":
		var addresses [1]string
		addresses[0] = r.URL.Query().Get("address")
		aMsg, err = kaspad.GetUTXOsByAddresses(addresses[:])
	case "/GetBlock":
		lBlock := r.URL.Query().Get("hash")
		aMsg, err = kaspad.GetBlock(lBlock, true)
	case "/GetBlockDAGInfo":
		aMsg, err = kaspad.GetBlockDAGInfo()
	case "/GetInfo":
		aMsg, err = kaspad.GetInfo()
	default:
		err = nil
		aMsg = &hRpcError{
			ErrorMsg: "No rpc procedure found for path " + r.URL.Path,
		}
	}
	if err != nil {
		aMsg = &hRpcError{
			ErrorMsg: err.Error(),
		}
	}
	rpcRetAny(w, r, aMsg)
}

func main() {
	var err error

	kaspad, err = rpcclient.NewRPCClient("localhost:16110")
	if err != nil {
		log.Printf("Error create RPCClient%s", err.Error())
		return
	}
	defer kaspad.Close()

	http.HandleFunc("/", rpcRetAll)

	log.Fatal(http.ListenAndServe(":16115", nil))

}
