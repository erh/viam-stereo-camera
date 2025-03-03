package flow

import (
	"context"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/golang/geo/r3"
	geo "github.com/kellydunn/golang-geo"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/movementsensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/spatialmath"

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
	Left       string
	Right      string
	FocalLengh float64 `json:"focal-length"`
}

func (cfg *Config) getFocalLength() float64 {
	if cfg.FocalLengh <= 0 {
		return 30.0
	}
	return cfg.FocalLengh
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

	dataLock   sync.Mutex
	linear     r3.Vector
	angular    spatialmath.AngularVelocity
	lastUpdate time.Time
	lastError  error
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

	go f.run()

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

func (f *flow) Position(ctx context.Context, extra map[string]interface{}) (*geo.Point, float64, error) {
	return nil, 0, movementsensor.ErrMethodUnimplementedPosition
}

func (f *flow) LinearVelocity(ctx context.Context, extra map[string]interface{}) (r3.Vector, error) {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	return f.linear, f.tooOld()
}

func (f *flow) AngularVelocity(ctx context.Context, extra map[string]interface{}) (spatialmath.AngularVelocity, error) {
	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	return f.angular, f.tooOld()
}

func (f *flow) LinearAcceleration(ctx context.Context, extra map[string]interface{}) (r3.Vector, error) {
	return r3.Vector{}, movementsensor.ErrMethodUnimplementedLinearAcceleration
}

func (f *flow) CompassHeading(ctx context.Context, extra map[string]interface{}) (float64, error) {
	return 0, movementsensor.ErrMethodUnimplementedCompassHeading
}

func (f *flow) Orientation(ctx context.Context, extra map[string]interface{}) (spatialmath.Orientation, error) {
	return nil, movementsensor.ErrMethodUnimplementedOrientation
}

func (f *flow) Properties(ctx context.Context, extra map[string]interface{}) (*movementsensor.Properties, error) {
	return &movementsensor.Properties{
		LinearVelocitySupported:  true,
		AngularVelocitySupported: true,
	}, nil

}

func (f *flow) Accuracy(ctx context.Context, extra map[string]interface{}) (*movementsensor.Accuracy, error) {
	return &movementsensor.Accuracy{}, nil
}

func (f *flow) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	res, err := movementsensor.DefaultAPIReadings(ctx, f, extra)
	if res != nil {
		res["lastUpdate"] = f.lastUpdate
		res["lastError"] = f.lastError
	}
	return res, err

}

func (f *flow) tooOld() error {
	if f.lastError != nil {
		return f.lastError
	}
	if time.Since(f.lastUpdate) > time.Second {
		return fmt.Errorf("no update since %v", f.lastUpdate)
	}
	return nil
}

type loopState struct {
	lastImage     image.Image
	lastImageTime time.Time
}

func (f *flow) doLoop(state *loopState) error {
	f.logger.Infof("starting loop")

	leftAll, meta, err := f.left.Images(f.cancelCtx)
	if err != nil {
		return err
	}
	if len(leftAll) == 0 {
		return fmt.Errorf("no images")
	}

	defer func() {
		state.lastImage = leftAll[0].Image
		state.lastImageTime = meta.CapturedAt
	}()

	diff := meta.CapturedAt.Sub(state.lastImageTime)
	if state.lastImage == nil || diff > time.Second {
		f.logger.Infof("no old image or too old")
		return nil
	}

	f.logger.Infof("starting flow computation")
	l, a, err := computeFlow(state.lastImage, leftAll[0].Image, diff, f.cfg.getFocalLength(), f.logger)
	if err != nil {
		f.logger.Infof("error computing flow")
		return err
	}

	f.logger.Infof("got %v %v", l, a)

	f.dataLock.Lock()
	defer f.dataLock.Unlock()
	f.linear = l
	f.angular = a
	f.lastUpdate = time.Now()

	return nil
}

func (f *flow) run() {
	state := loopState{}
	for f.cancelCtx.Err() == nil {
		f.lastError = f.doLoop(&state)
		if f.lastError != nil {
			f.lastUpdate = time.Now()
		}
		time.Sleep(500 * time.Millisecond)
	}
}
