package utils

import (
	"image"
)

func GetFirstOpaquePixelY(image image.Image) uint16 {
	bounds := image.Bounds()
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := image.At(x, y).RGBA()
			if a > 0 {
				return uint16(y)
			}
		}
	}
	return 0
}
