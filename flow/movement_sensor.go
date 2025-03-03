package flow

import (
	"context"
	"fmt"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/movementsensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"viamstereocamera"
)

var Model = viamstereocamera.NamespaceFamily.WithModel("flow-movement-sensor")


func init() {
	resource.RegisterComponent(movementsensor.API, Model,
		resource.Registration[movementsensor.MovementSensor, *Config]{
			Constructor: newFlow,
		},
	)
}

type Config struct {
	Left string
	Right string
}

func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Left == "" {
		return nil, fmt.Errorf("need left")
	}
	if cfg.Right == "" {
		return nil, fmt.Errorf("need right")
	}

	return []string{cfg.Left, cfg.Right}, nil
}

type flow struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	left, right camera.Camera
}

func newFlow(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (movementsensor.MovementSensor, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewFlow(ctx, deps, rawConf.ResourceName(), conf, logger)

}

func NewFlow(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (movementsensor.MovementSensor, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	
	f := &flow{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	var err error
	f.left, err = camera.FromDependencies(deps, conf.Left)
	if err != nil {
		return nil, err
	}
	f.right, err = camera.FromDependencies(deps, conf.Right)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (f *flow) Name() resource.Name {
	return f.name
}


func (f *flow) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (f *flow) Close(context.Context) error {
	f.cancelFunc()
	return nil
}

