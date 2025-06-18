package connection

import (
	"sync"

	"github.com/Phillezi/tunman-remaster/internal/defaults"
	"github.com/Phillezi/tunman-remaster/pkg/controller"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/Phillezi/tunman-remaster/utils"
	"go.uber.org/zap"
)

var (
	once     sync.Once
	instance ctrlpb.TunnelServiceClient
)

func C() ctrlpb.TunnelServiceClient {
	once.Do(func() {
		ctrl, err := controller.Dial(utils.Or(defaults.SocketPath))
		if err != nil {
			zap.L().Error("failed to connect", zap.Error(err))
			return
		}
		instance = ctrl
	})
	return instance
}
