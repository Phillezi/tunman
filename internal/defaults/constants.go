package defaults

const (
	LockPath           string = "/tmp/tunmand.lock"
	SocketPath         string = "unix:/tmp/tunmand.sock" //"unix:/var/run/tunmand.sock"
	DefaultPublishHost string = "0.0.0.0"
	DefaultDBPath      string = "./state.db"
)
