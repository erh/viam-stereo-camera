package viamstereocamera

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/golang/geo/r3"
	
	"go.viam.com/rdk/pointcloud"
)


// StereoPCDConfig holds the configuration parameters for stereo to point cloud conversion
type StereoPCDConfig struct {
	
	Baseline float64 // distance between cameras in meters
	FocalLength float64 // focal length of the camera in pixels
	
	MinDisparity float64
	MaxDisparity float64
	
	DisparityStep int // controls how many pixels to skip when comparing (higher = faster but less dense)
	PixelStep int // controls how many pixels to skip in the image (higher = faster but less dense)
}

func calculatePixelDifference(img1, img2 image.Image, x1, y1, x2, y2 int) float64 {
	// Simple SAD (Sum of Absolute Differences) for the RGB values
	r1, g1, b1, _ := img1.At(x1, y1).RGBA()
	r2, g2, b2, _ := img2.At(x2, y2).RGBA()
	
	rDiff := math.Abs(float64(r1) - float64(r2))
	gDiff := math.Abs(float64(g1) - float64(g2))
	bDiff := math.Abs(float64(b1) - float64(b2))
	
	return rDiff + gDiff + bDiff
}

func StereoToPointCloud(leftImg, rightImg image.Image, config StereoPCDConfig) (pointcloud.PointCloud, error) {
	bounds := leftImg.Bounds()
	rightBounds := rightImg.Bounds()
	
	// Check if images have the same dimensions
	if bounds.Dx() != rightBounds.Dx() || bounds.Dy() != rightBounds.Dy() {
		return nil, fmt.Errorf("images must have the same dimensions")
	}
	
	// Calculate the principal point (usually the center of the image)
	cx := float64(bounds.Dx()) / 2.0
	cy := float64(bounds.Dy()) / 2.0
	
	// Create a new point cloud
	pc := pointcloud.New()
	
	// For each pixel in the left image
	for y := bounds.Min.Y; y < bounds.Max.Y; y += config.PixelStep {
		for x := bounds.Min.X; x < bounds.Max.X; x += config.PixelStep {
			// Get the RGB color of the current pixel
			r, g, b, _ := leftImg.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			
			// Find the best matching pixel in the right image (along the epipolar line)
			bestDisparity := 0.0
			minDiff := math.MaxFloat64
			
			maxSearchX := x
			if x-int(config.MaxDisparity) > bounds.Min.X {
				maxSearchX = x - int(config.MaxDisparity)
			}
			
			for searchX := x; searchX >= maxSearchX; searchX -= config.DisparityStep {
				// Calculate the sum of absolute differences (SAD) for a small window
				diff := calculatePixelDifference(leftImg, rightImg, x, y, searchX, y)
				
				if diff < minDiff {
					minDiff = diff
					bestDisparity = float64(x - searchX)
				}
			}
			
			// Filter out low confidence disparity values
			if bestDisparity > config.MinDisparity && bestDisparity < config.MaxDisparity {
				// Calculate Z (depth) using the formula: Z = (baseline * focal_length) / disparity
				z := (config.Baseline * config.FocalLength) / bestDisparity
				
				// Calculate X and Y using the pinhole camera model
				x3d := ((float64(x) - cx) * z) / config.FocalLength
				y3d := ((float64(y) - cy) * z) / config.FocalLength
				
				
				err := pc.Set(
					r3.Vector{X: x3d, Y: y3d, Z: z},
					pointcloud.NewColoredData(color.NRGBA{R: r8, G: g8, B: b8, A:1}),
				)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	
	return pc, nil
}
