package main

import (
	"time"

	"github.com/Phillezi/tunman-remaster/internal/defaults"
	"github.com/Phillezi/tunman-remaster/interrupt"
	"github.com/Phillezi/tunman-remaster/log"
	"github.com/Phillezi/tunman-remaster/pkg/controller"
	"github.com/Phillezi/tunman-remaster/pkg/manager"
	"github.com/Phillezi/tunman-remaster/utils"
	"go.uber.org/zap"
)

func init() {
	log.Setup()
	interrupt.GetInstance().AddShutdownHook(func() { zap.L().Info("daemon shutdown") })
}

func main() {
	zap.L().Info("start")
	defer interrupt.GetInstance().Shutdown()

	man := manager.New()
	interrupt.GetInstance().AddShutdownHook(func() { zap.L().Info("manager shutdown"); man.Shutdown() })

	go func() {
		if err := controller.ListenAndServe(utils.Or(defaults.SocketPath), man, nil); err != nil {
			zap.L().Error("error serving", zap.Error(err))
		}
	}()

	<-interrupt.GetInstance().Context().Done()
	interrupt.GetInstance().Wait(5 * time.Second)
}
