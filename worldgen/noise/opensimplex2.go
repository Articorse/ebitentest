// Port of the 2D noise from KdotJPG's OpenSimplex2 (the fast variant), which is public domain:
// https://github.com/KdotJPG/OpenSimplex2
package noise

import "math"

const (
	primeX         = 0x5205402B9270C86F
	primeY         = 0x598CD327003817B5
	hashMultiplier = 0x53A3F72DEEC546F5

	skew2D        = 0.366025403784439
	unskew2D      = -0.21132486540518713
	rSquared2D    = 0.5
	normalizer2D  = 0.01001634121365712
	nGradsExp2D   = 7
	nGrads2D      = 1 << nGradsExp2D
	gradIdxMask2D = (nGrads2D - 1) << 1
)

var gradients2D = func() [nGrads2D * 2]float64 {
	grads := []float64{
		0.38268343236509, 0.923879532511287,
		0.923879532511287, 0.38268343236509,
		0.923879532511287, -0.38268343236509,
		0.38268343236509, -0.923879532511287,
		-0.38268343236509, -0.923879532511287,
		-0.923879532511287, -0.38268343236509,
		-0.923879532511287, 0.38268343236509,
		-0.38268343236509, 0.923879532511287,
		0.130526192220052, 0.99144486137381,
		0.608761429008721, 0.793353340291235,
		0.793353340291235, 0.608761429008721,
		0.99144486137381, 0.130526192220051,
		0.99144486137381, -0.130526192220051,
		0.793353340291235, -0.60876142900872,
		0.608761429008721, -0.793353340291235,
		0.130526192220052, -0.99144486137381,
		-0.130526192220052, -0.99144486137381,
		-0.608761429008721, -0.793353340291235,
		-0.793353340291235, -0.608761429008721,
		-0.99144486137381, -0.130526192220052,
		-0.99144486137381, 0.130526192220051,
		-0.793353340291235, 0.608761429008721,
		-0.608761429008721, 0.793353340291235,
		-0.130526192220052, 0.99144486137381,
	}

	var table [nGrads2D * 2]float64
	for i := range table {
		table[i] = grads[i%len(grads)] / normalizer2D
	}
	return table
}()

// Smooth noise in about [-1, 1], with features roughly one unit across
func OpenSimplex2(seed int64, x, y float64) float64 {
	s := skew2D * (x + y)
	xs, ys := x+s, y+s

	xsb, ysb := math.Floor(xs), math.Floor(ys)
	xi, yi := xs-xsb, ys-ysb
	xsbp, ysbp := int64(xsb)*primeX, int64(ysb)*primeY

	t := (xi + yi) * unskew2D
	dx0, dy0 := xi+t, yi+t

	value := 0.0
	if a0 := rSquared2D - dx0*dx0 - dy0*dy0; a0 > 0 {
		value += (a0 * a0) * (a0 * a0) * grad(seed, xsbp, ysbp, dx0, dy0)
	}

	a1 := (2*(1+2*unskew2D)*(1/unskew2D+2))*t + (-2*(1+2*unskew2D)*(1+2*unskew2D) + (rSquared2D - dx0*dx0 - dy0*dy0))
	if a1 > 0 {
		dx1, dy1 := dx0-(1+2*unskew2D), dy0-(1+2*unskew2D)
		value += (a1 * a1) * (a1 * a1) * grad(seed, xsbp+primeX, ysbp+primeY, dx1, dy1)
	}

	if dy0 > dx0 {
		dx2, dy2 := dx0-unskew2D, dy0-(unskew2D+1)
		if a2 := rSquared2D - dx2*dx2 - dy2*dy2; a2 > 0 {
			value += (a2 * a2) * (a2 * a2) * grad(seed, xsbp, ysbp+primeY, dx2, dy2)
		}
	} else {
		dx2, dy2 := dx0-(unskew2D+1), dy0-unskew2D
		if a2 := rSquared2D - dx2*dx2 - dy2*dy2; a2 > 0 {
			value += (a2 * a2) * (a2 * a2) * grad(seed, xsbp+primeX, ysbp, dx2, dy2)
		}
	}

	return value
}

func grad(seed, xsvp, ysvp int64, dx, dy float64) float64 {
	hash := (seed ^ xsvp ^ ysvp) * hashMultiplier
	hash ^= hash >> (64 - nGradsExp2D + 1)
	gi := int(hash) & gradIdxMask2D
	return gradients2D[gi]*dx + gradients2D[gi|1]*dy
}
