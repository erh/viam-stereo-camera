package viamstereocamera

import (
	"context"
	"errors"
	"fmt"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
)

var (
	StereoCamera     = NamespaceFamily.WithModel("stereo-camera")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(camera.API, StereoCamera,
		resource.Registration[camera.Camera, *Config]{
			Constructor: newViamStereoCameraStereoCamera,
		},
	)
}

type Config struct {
	Left  string
	Right string

	DistanceMeters    float64 `json:"distance-meters"`
	FocalLengthPixels float64 `json:"focal-length-pixels"`

	MinDisparity float64 `json:"max-disparity"`
	MaxDisparity float64 `json:"min-disparity"`

	// DisparityStep controls how many pixels to skip when comparing (higher = faster but less dense)
	DisparityStep int

	// PixelStep controls how many pixels to skip in the image (higher = faster but less dense)
	PixelStep int
}

func (cfg *Config) getMinDisparity() float64 {
	if cfg.MinDisparity <= 0 {
		return 1
	}
	return cfg.MinDisparity
}

func (cfg *Config) getMaxDisparity() float64 {
	if cfg.MaxDisparity <= 0 {
		return 64
	}
	return cfg.MaxDisparity
}

func (cfg *Config) getDisparityStep() int {
	return 1
}

func (cfg *Config) getPixelStep() int {
	return 1
}

func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Left == "" {
		return nil, fmt.Errorf("need left")
	}
	if cfg.Right == "" {
		return nil, fmt.Errorf("need right")
	}

	if cfg.DistanceMeters <= 0 {
		return nil, fmt.Errorf("need distance-meters")
	}

	if cfg.FocalLengthPixels <= 0 {
		return nil, fmt.Errorf("need focal-length-pixels")
	}

	return []string{cfg.Left, cfg.Right}, nil
}

type viamStereoCameraStereoCamera struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	left, right camera.Camera
}

func newViamStereoCameraStereoCamera(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (camera.Camera, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewStereoCamera(ctx, deps, rawConf.ResourceName(), conf, logger)

}

func NewStereoCamera(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (camera.Camera, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &viamStereoCameraStereoCamera{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	var err error
	s.left, err = camera.FromDependencies(deps, conf.Left)
	if err != nil {
		return nil, err
	}
	s.right, err = camera.FromDependencies(deps, conf.Right)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *viamStereoCameraStereoCamera) Name() resource.Name {
	return s.name
}

func (s *viamStereoCameraStereoCamera) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (s *viamStereoCameraStereoCamera) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}

func (s *viamStereoCameraStereoCamera) Image(ctx context.Context, mimeType string, extra map[string]interface{}) ([]byte, camera.ImageMetadata, error) {
	return s.left.Image(ctx, mimeType, extra)
}

func (s *viamStereoCameraStereoCamera) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	return s.left.Images(ctx)
}

func (s *viamStereoCameraStereoCamera) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	// TODO:  parallelize image loading

	leftAll, _, err := s.left.Images(ctx)
	if err != nil {
		return nil, err
	}

	rightAll, _, err := s.right.Images(ctx)
	if err != nil {
		return nil, err
	}

	if len(leftAll) != 1 {
		return nil, fmt.Errorf("why is leftAll %d", len(leftAll))
	}

	if len(rightAll) != 1 {
		return nil, fmt.Errorf("why is rightAll %d", len(rightAll))
	}

	c := StereoPCDConfig{
		Baseline:    s.cfg.DistanceMeters,
		FocalLength: s.cfg.FocalLengthPixels,

		MinDisparity: s.cfg.getMinDisparity(),
		MaxDisparity: s.cfg.getMaxDisparity(),

		DisparityStep: s.cfg.getDisparityStep(),
		PixelStep:     s.cfg.getPixelStep(),
	}
	return StereoToPointCloud(leftAll[0].Image, rightAll[0].Image, c)
}

func (s *viamStereoCameraStereoCamera) Properties(ctx context.Context) (camera.Properties, error) {
	return camera.Properties{
		SupportsPCD: true,
	}, nil
}
