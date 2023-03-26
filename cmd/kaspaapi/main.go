package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/godror/godror"
)

var dbOra *sql.DB

func dbExecApi(method string, jobtext string) (string, error) {
	var pOraOut string

	StmInsJob, err := dbOra.Prepare("BEGIN IBS.Z$GA_LIB_LPOOL_RO.EXECPROC(:pCoin, :pMethod, :pInpPost, :pOutPost); END;")
	if err != nil {
		return "{}", err
	}

	_, err = StmInsJob.Exec(
		sql.Named("pCoin", "kaspa"),
		sql.Named("pMethod", method),
		sql.Named("pInpPost", jobtext),
		sql.Named("pOutPost", sql.Out{Dest: &pOraOut}),
	)
	StmInsJob.Close()
	if err != nil {
		return "{}", err
	}

	return pOraOut, nil
}

func rpcRetStr(w http.ResponseWriter, r *http.Request, rStr string) {
	w.Header().Add("Content-Type", "application/json")
	w.Write([]byte(rStr))
}

func rpcRetAll(w http.ResponseWriter, r *http.Request) {
	var sMsg string
	var err error
	switch r.URL.Path {
	case "/stats":
		sMsg, err = dbExecApi("stats", "")
	default:
		err = nil
		sMsg = `{"error":"No rpc procedure found for path ` + r.URL.Path + `"}`
	}
	if err != nil {
		sMsg = `{"error":"` + err.Error() + `"}`
	}
	rpcRetStr(w, r, sMsg)
}

func main() {
	db, err := sql.Open("godror", `user="POOLAPI" password="figAn3Write+2" connectString="cft3" noTimezoneCheck=true`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	dbOra = db

	http.HandleFunc("/", rpcRetAll)

	log.Fatal(http.ListenAndServe(":16116", nil))

}
