package flow

import (
	"errors"
	"image"
	"math"
	"time"

	"github.com/golang/geo/r3"
	"gocv.io/x/gocv"

	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/spatialmath"
)

// computeFlow calculates linear and angular velocity from two consecutive images
// prev: previous image frame
// now: current image frame
// timeBetween: time duration between the two frames
// Returns:
// - linear velocity as r3.Vector (x,y,z components in units/second)
// - angular velocity using Viam's spatialmath.AngularVelocity
// - error if processing fails
func computeFlow(prev, now image.Image, timeBetween time.Duration, focalLengthPx float64, logger logging.Logger) (r3.Vector, spatialmath.AngularVelocity, error) {
	// Convert time to seconds
	dt := timeBetween.Seconds()
	if dt <= 0 {
		return r3.Vector{}, spatialmath.AngularVelocity{}, errors.New("time between frames must be positive")
	}

	// Convert image.Image to gocv.Mat
	prevMat, err := imageToMat(prev)
	if err != nil {
		return r3.Vector{}, spatialmath.AngularVelocity{}, err
	}
	defer prevMat.Close()

	nowMat, err := imageToMat(now)
	if err != nil {
		return r3.Vector{}, spatialmath.AngularVelocity{}, err
	}
	defer nowMat.Close()

	// Convert to grayscale for optical flow

	prevGray := gocv.NewMat()
	defer prevGray.Close()
	gocv.CvtColor(prevMat, &prevGray, gocv.ColorBGRToGray)

	nowGray := gocv.NewMat()
	defer nowGray.Close()
	gocv.CvtColor(nowMat, &nowGray, gocv.ColorBGRToGray)

	// Find features to track in previous image usingn Shi-Tomasi corner detector

	prevPts := gocv.NewMat()
	defer prevPts.Close()
	gocv.GoodFeaturesToTrack(prevGray, &prevPts, 100, 0.3, 10)

	// If no features found, return zero velocity
	if prevPts.Rows() == 0 {
		return r3.Vector{}, spatialmath.AngularVelocity{}, nil
	}

	// Calculate optical flow using Lucas-Kanade method
	nextPts := gocv.NewMat()
	defer nextPts.Close()

	status := gocv.NewMat()
	defer status.Close()

	errMat := gocv.NewMat()
	defer errMat.Close()

	// Apply optical flow algorithm
	gocv.CalcOpticalFlowPyrLK(prevGray, nowGray, prevPts, nextPts, &status, &errMat)

	// now we do our math in vectors

	nextPtsVec := gocv.NewPointVectorFromMat(nextPts)
	defer nextPtsVec.Close()

	prevPtsVec := gocv.NewPointVectorFromMat(prevPts)
	defer prevPtsVec.Close()

	logger.Debugf("prev/next pt vectors %d %d", prevPtsVec.Size(), nextPtsVec.Size())

	// Process optical flow results
	var sumDx, sumDy float64
	var sumAngularZ float64
	validPoints := 0

	// Get image center for angular calculations
	centerX := float64(prevGray.Cols()) / 2
	centerY := float64(prevGray.Rows()) / 2

	for i := 0; i < prevPtsVec.Size(); i++ {
		// Check if point was successfully tracked
		if status.GetUCharAt(i, 0) == 1 {
			prevPt := prevPtsVec.At(i)
			nextPt := nextPtsVec.At(i)

			// Calculate displacement
			dx := float64(nextPt.X - prevPt.X)
			dy := float64(nextPt.Y - prevPt.Y)

			// Calculate angular displacement around z-axis using points relative to center
			prevAngle := math.Atan2(float64(prevPt.Y)-centerY, float64(prevPt.X)-centerX)
			nextAngle := math.Atan2(float64(nextPt.Y)-centerY, float64(nextPt.X)-centerX)
			angularDisplacement := normalizeAngle(nextAngle - prevAngle)

			sumDx += dx
			sumDy += dy
			sumAngularZ += angularDisplacement
			validPoints++
		}
	}

	if validPoints == 0 {
		return r3.Vector{}, spatialmath.AngularVelocity{}, nil
	}

	// Average displacements
	avgDx := sumDx / float64(validPoints)
	avgDy := sumDy / float64(validPoints)
	avgAngularZ := sumAngularZ / float64(validPoints)

	// Calculate velocities
	linearVelX := avgDx / dt
	linearVelY := avgDy / dt
	angularVelZ := avgAngularZ / dt

	// Convert from pixel space to physical space (approximation)
	// In real usage, this would depend on proper camera calibration
	// This is a simplification using an estimated focal length
	linearVelX = linearVelX / focalLengthPx
	linearVelY = linearVelY / focalLengthPx

	return r3.Vector{X: linearVelX, Y: linearVelY}, spatialmath.AngularVelocity{Z: angularVelZ}, nil
}

// Helper function to normalize angle to [-π, π]
func normalizeAngle(angle float64) float64 {
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

// Helper function to convert image.Image to gocv.Mat
func imageToMat(img image.Image) (gocv.Mat, error) {
	// Convert image.Image to gocv.Mat
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Create new Mat with appropriate size and type
	mat := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)

	// Copy pixel data
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			// Convert from 0-65535 to 0-255
			mat.SetUCharAt(y, x*3, uint8(b>>8))
			mat.SetUCharAt(y, x*3+1, uint8(g>>8))
			mat.SetUCharAt(y, x*3+2, uint8(r>>8))
		}
	}

	return mat, nil
}
