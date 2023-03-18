package kaspastratum

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/godror/godror"
	"go.uber.org/zap"
)

type OraJob struct {
	jobType string
	jobText string
}

type OraBackend struct {
	Jobs       map[uint32]OraJob
	JobLock    sync.RWMutex
	FileLock   sync.RWMutex
	JobCounter uint32
	JobDeleted uint32
	Coin       string
	Db         *sql.DB
	//StmInsJob  *sql.Stmt
	logger *zap.SugaredLogger
}

func NewOraBackend(coin string, logger *zap.SugaredLogger, connstr string) (*OraBackend, error) {
	db, err := sql.Open("godror", connstr)
	if err != nil {
		return nil, err
	}

	//stmInsJob, err := db.Prepare("BE GIN IBS.Z$GA_LIB_LPOOL_RW.EXECPROC(:pCoin, :pMethod, :pInpPost, :pOutPost); END;")
	//if err != nil {
	//	return nil, err
	//}

	retOb := &OraBackend{
		Jobs:       make(map[uint32]OraJob),
		JobLock:    sync.RWMutex{},
		FileLock:   sync.RWMutex{},
		JobCounter: 0,
		JobDeleted: 0,
		Coin:       coin,
		Db:         db,
		//StmInsJob:  stmInsJob,
		logger: logger.With(zap.String("component", "OraBackend")),
	}

	go retOb.startPushThread()

	return retOb, nil
}

func (ob *OraBackend) AddObj(jobType string, jobObj any) {
	barr, err := json.Marshal(jobObj)

	if err != nil {
		ob.logger.Errorf("Error Marshaling OraBackend: %s", err.Error())
		return
	}

	ob.AddJob(jobType, string(barr))
}

func (ob *OraBackend) AddJob(jobType string, jobText string) {
	ob.JobLock.Lock()

	newJobCounter := ob.JobCounter
	newJobCounter++
	ob.Jobs[newJobCounter] = OraJob{
		jobType: jobType,
		jobText: jobText,
	}
	ob.JobCounter = newJobCounter
	ob.JobLock.Unlock()
}

type ErrorJob struct {
	Coin      string
	Method    string
	JobType   string
	JobText   string
	ErrorText string
}

func (ob *OraBackend) writeJobtoFile(errjob ErrorJob) {
	ob.FileLock.Lock()

	logFileName := "errdata_" + time.Now().Format("2006-01-02T15") + ".txt"

	barr, err := json.Marshal(errjob)
	if err != nil {
		ob.logger.Errorf("Error marshaling error task: %s", err.Error())
		return
	}

	efile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		ob.logger.Errorf("Error open file for error task: %s", err.Error())
		return
	}

	_, err = fmt.Fprintf(efile, "%s\n", string(barr))
	if err != nil {
		ob.logger.Errorf("Error write error task to file: %s", err.Error())
		return
	}

	efile.Close()
	ob.FileLock.Unlock()
}

func (ob *OraBackend) startPushThread() error {
	for {
		// console formatting is terrible. Good luck whever touches anything
		time.Sleep(1 * time.Second)
		for ob.JobCounter > ob.JobDeleted {
			ob.JobDeleted++
			ob.JobLock.Lock()
			curJob := ob.Jobs[ob.JobDeleted]
			ob.JobLock.Unlock()

			var pOraOut string

			StmInsJob, err := ob.Db.Prepare("BEGIN IBS.Z$GA_LIB_LPOOL_RW.EXECPROC(:pCoin, :pMethod, :pInpPost, :pOutPost); END;")

			if err == nil {
				_, err = StmInsJob.Exec(
					sql.Named("pCoin", ob.Coin),
					sql.Named("pMethod", curJob.jobType),
					sql.Named("pInpPost", curJob.jobText),
					sql.Named("pOutPost", sql.Out{Dest: &pOraOut}),
				)
				StmInsJob.Close()
			}

			// If StmInsJob return non-empty string to pOraOut, it's error too
			if err == nil {
				if pOraOut != "" {
					err = errors.New(pOraOut)
				}
			}

			if err != nil {
				ob.writeJobtoFile(ErrorJob{
					Coin:      ob.Coin,
					JobType:   curJob.jobType,
					JobText:   curJob.jobText,
					ErrorText: err.Error(),
				})
			}

			ob.JobLock.Lock()
			delete(ob.Jobs, ob.JobDeleted)
			ob.JobLock.Unlock()

		}
	}
}
