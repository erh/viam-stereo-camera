package flow

import (
	"image"
	_ "image/jpeg"
	"math"
	"os"
	"testing"
	"time"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func read(fn string) (image.Image, error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

func TestFlow1(t *testing.T) {
	logger := logging.NewTestLogger(t)

	a1, err := read("data/pa1.jpg")
	test.That(t, err, test.ShouldBeNil)

	a2, err := read("data/pa2.jpg")
	test.That(t, err, test.ShouldBeNil)

	focalLength := 30.0

	l, a, err := computeFlow(a2, a1, time.Second, focalLength, logger)
	test.That(t, err, test.ShouldBeNil)

	logger.Infof("hi %v %v", l, a)

	test.That(t, l.Z, test.ShouldAlmostEqual, 0)
	test.That(t, l.Y, test.ShouldBeGreaterThan, 0)

	ratio := math.Abs(l.Y) / math.Abs(l.X)
	test.That(t, ratio, test.ShouldBeGreaterThan, 10) // really more, sanity check

	test.That(t, l.Y, test.ShouldAlmostEqual, 1, .1)

	test.That(t, math.Abs(a.Z), test.ShouldBeGreaterThan, 0)
	test.That(t, a.Z, test.ShouldAlmostEqual, 0, .1)

}
