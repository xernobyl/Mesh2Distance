/*
A simple math vector library... New functions added as needed.
*/

package vec

import (
	"github.com/chewxy/math32"
	"golang.org/x/exp/constraints"
)

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

func Length(a Vec3) float32 {
	return math32.Sqrt(Dot2(a))
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

func Clamp[T constraints.Ordered](v, min, max T) T {
	if v > max {
		return max
	}

	if v < min {
		return min
	}

	return v
}

func Saturate[T ~float32 | ~float64](v T) T {
	return Clamp(v, T(0.0), T(1.0))
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min3[T constraints.Ordered](a, b, c T) T {
	return Min(a, Min(b, c))
}

func Max3[T constraints.Ordered](a, b, c T) T {
	return Max(a, Max(b, c))
}

func MinN[T constraints.Ordered](nums ...T) T {
	if len(nums) == 0 {
		panic("Min requires at least one argument")
	}

	minValue := nums[0]
	for _, num := range nums[1:] {
		if num < minValue {
			minValue = num
		}
	}

	return minValue
}

func MaxN[T constraints.Ordered](nums ...T) T {
	if len(nums) == 0 {
		panic("Min requires at least one argument")
	}

	maxValue := nums[0]
	for _, num := range nums[1:] {
		if num > maxValue {
			maxValue = num
		}
	}

	return maxValue
}
