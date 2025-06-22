package connection

import (
	"sync"

	"github.com/Phillezi/tunman/internal/defaults"
	"github.com/Phillezi/tunman/pkg/controller"
	ctrlpb "github.com/Phillezi/tunman/proto"
	"github.com/Phillezi/tunman/utils"
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
