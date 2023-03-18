package kaspastratum

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/galiy/kaspa-pool/src/gostratum"
	"github.com/google/uuid"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// const version = "v1.1.6"
const minBlockWaitTime = 500 * time.Millisecond

type BridgeConfig struct {
	StratumPort    string        `yaml:"stratum_port"`
	RPCServer      string        `yaml:"kaspad_address"`
	UseLogFile     bool          `yaml:"log_to_file"`
	BlockWaitTime  time.Duration `yaml:"block_wait_time"`
	MinShareDiff   uint          `yaml:"min_share_diff"`
	ExtranonceSize uint          `yaml:"extranonce_size"`
	PoolWallet     string        `yaml:"pool_wallet"`
	OraConnStr     string        `yaml:"ora_connstr"`
}

func configureZap(cfg BridgeConfig) (*zap.SugaredLogger, func()) {
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.RFC3339TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(pe)
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	if !cfg.UseLogFile {
		return zap.New(zapcore.NewCore(consoleEncoder,
			zapcore.AddSync(colorable.NewColorableStdout()), zap.InfoLevel)).Sugar(), func() {}
	}

	// log file fun
	logFile, err := os.OpenFile("bridge.log", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zap.InfoLevel),
	)
	return zap.New(core).Sugar(), func() { logFile.Close() }
}

type AuthDbQuery struct {
	CurrentTime string
	SesUid      string
	RemoteAddr  string
	WalletAddr  string
	WorkerName  string
	Password    string
	RemoteApp   string
}

func ListenAndServe(cfg BridgeConfig) error {
	logger, logCleanup := configureZap(cfg)
	defer logCleanup()

	blockWaitTime := cfg.BlockWaitTime
	if blockWaitTime < minBlockWaitTime {
		blockWaitTime = minBlockWaitTime
	}

	// Create RPC node Client
	ksApi, err := NewKaspaAPI(cfg.RPCServer, blockWaitTime, logger, cfg.PoolWallet)
	if err != nil {
		return err
	}

	// Create Oracle backend
	backend, err := NewOraBackend("kaspa", logger, cfg.OraConnStr)
	if err != nil {
		return err
	}

	// Create share handler object
	shareHandler := newShareHandler(ksApi.kaspad, backend)
	minDiff := cfg.MinShareDiff
	if minDiff < 1 {
		minDiff = 1
	}

	extranonceSize := cfg.ExtranonceSize
	if extranonceSize > 3 {
		extranonceSize = 3
	}

	// Create client handler
	clientHandler := newClientListener(logger, shareHandler, float64(minDiff), int8(extranonceSize))

	handlers := gostratum.DefaultHandlers()
	// override the submit handler with an actual useful handler
	handlers[string(gostratum.StratumMethodSubmit)] =
		func(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
			if err := shareHandler.HandleSubmit(ctx, event); err != nil {
				ctx.Logger.Sugar().Error(err) // sink error
			}
			return nil
		}

	handlers[string(gostratum.StratumMethodAuthorize)] =
		func(ctx *gostratum.StratumContext, event gostratum.JsonRpcEvent) error {
			if err := gostratum.HandleAuthorize(ctx, event); err != nil {
				return err
			}

			password, ok := event.Params[1].(string)
			if !ok {
				password = ""
			}
			ctx.Password = password
			ctx.SesUid = strings.Replace(uuid.New().String(), "-", "", -1)

			ctx.Logger.Info(fmt.Sprintf("client authorized, address: %s password: %s uid: %s", ctx.WalletAddr, ctx.Password, ctx.SesUid))

			backend.AddObj("session", AuthDbQuery{
				CurrentTime: time.Now().Format(time.RFC3339Nano),
				SesUid:      ctx.SesUid,
				RemoteAddr:  ctx.RemoteAddr,
				WalletAddr:  ctx.WalletAddr,
				WorkerName:  ctx.WorkerName,
				Password:    ctx.Password,
				RemoteApp:   ctx.RemoteApp,
			})

			return nil
		}

	stratumConfig := gostratum.StratumListenerConfig{
		Port:           cfg.StratumPort,
		HandlerMap:     handlers,
		StateGenerator: MiningStateGenerator,
		ClientListener: clientHandler,
		Logger:         logger.Desugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ksApi.Start(ctx, func() {
		clientHandler.NewBlockAvailable(ksApi)
	})

	//if cfg.PrintStats {
	//	go shareHandler.startStatsThread()
	//}

	return gostratum.NewListener(stratumConfig).Listen(context.Background())
}
