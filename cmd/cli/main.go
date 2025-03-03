package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"os"
	"strconv"

	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"

	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/spatialmath"

	
	"viamstereocamera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/components/arm"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func realMain() error {
	ctx := context.Background()
	logger := logging.NewLogger("cli")

	deps := resource.Dependencies{}
	// can load these from a remote machine if you need

	cfg := viamstereocamera.Config{}

	thing, err := viamstereocamera.NewStereoCamera(ctx, deps, arm.Named("foo"), &cfg, logger)
	if err != nil {
		return err
	}
	defer thing.Close(ctx)

	return nil
}
