package main

import (
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/movementsensor"

	"viamstereocamera"
	"viamstereocamera/flow"
)

func main() {
	// ModularMain can take multiple APIModel arguments, if your module implements multiple models.
	module.ModularMain(
		resource.APIModel{ camera.API, viamstereocamera.StereoCamera},
		resource.APIModel{ movementsensor.API, flow.Model},
	)
}
