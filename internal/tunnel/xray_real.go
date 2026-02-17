//go:build xray

package tunnel

import (
	"bytes"
	"fmt"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func init() {
	defaultLoader = realLoader
}

// realLoader uses xray-core library to create a real tunnel instance.
func realLoader(jsonCfg []byte) (XrayInstance, error) {
	pbConfig, err := core.LoadConfig("json", bytes.NewReader(jsonCfg))
	if err != nil {
		return nil, fmt.Errorf("xray LoadConfig: %w", err)
	}

	instance, err := core.New(pbConfig)
	if err != nil {
		return nil, fmt.Errorf("xray core.New: %w", err)
	}

	return &realXrayInstance{instance: instance}, nil
}

type realXrayInstance struct {
	instance *core.Instance
}

func (r *realXrayInstance) Start() error {
	return r.instance.Start()
}

func (r *realXrayInstance) Close() error {
	return r.instance.Close()
}
