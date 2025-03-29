/*
A simple math vector library... New functions added as needed.
*/

package main

import "github.com/chewxy/math32"

type Vec3 [3]float32

func Add(a, b Vec3) Vec3 {
	return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func Sub(a, b Vec3) Vec3 {
	return Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func Dot(a, b Vec3) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func Mul(a, b Vec3) Vec3 {
	return Vec3{a[0] * b[0], a[1] * b[1], a[2] * b[2]}
}

func Scale(a Vec3, b float32) Vec3 {
	return Vec3{a[0] * b, a[1] * b, a[2] * b}
}

func Dot2(a Vec3) float32 {
	return a[0]*a[0] + a[1]*a[1] + a[2]*a[2]
}

func Cross(a, b Vec3) Vec3 {
	return Vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func Sign(a float32) float32 {
	if a > 0.0 {
		return 1.0
	}

	if a < 0.0 {
		return -1.0
	}

	return 0.0
}

func Normalize(a Vec3) Vec3 {
	l := math32.Sqrt(a[0]*a[0] + a[1]*a[1] + a[2]*a[2])
	return Vec3{a[0] / l, a[1] / l, a[2] / l}
}

func Clamp(v, min, max float32) float32 {
	if v > max {
		return max
	}

	if v < min {
		return min
	}

	return v
}

func Saturate(v float32) float32 {
	return Clamp(v, 0.0, 1.0)
}

func Min(a, b float32) float32 {
	if a < b {
		return a
	}

	return b
}

func Min3(a, b, c float32) float32 {
	return Min(a, Min(b, c))
}
