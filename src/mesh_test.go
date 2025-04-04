package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xernobyl/mesh2distance/src/vec"
)

func TestDistance(t *testing.T) {
	a := vec.Vec3{0.0, 1.0, -1.0}
	b := vec.Vec3{0.0, -1.0, -1.0}
	c := vec.Vec3{0.0, 0.0, 1.0}

	p := vec.Vec3{-1.0, 0, 0}
	d0 := distance(p, a, b, c)
	d1 := distance(p, b, a, c)

	assert.Equal(t, -d0, d1) // distance should be symmetric
	assert.Equal(t, float32(-1.095445), d0)

	p = vec.Vec3{0.0, 1.0, -1.0}
	d0 = distance(p, b, a, c)
	assert.Equal(t, float32(0.0), d0)

	p = vec.Vec3{0.0, 3.0, -1.0}
	d0 = distance(p, b, a, c)
	assert.Equal(t, float32(2.0), d0)
}

func TestLoadOBJ(t *testing.T) {
	mesh, err := LoadOBJ("../tetrahedron.obj")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(mesh.Triangles))
	assert.Equal(t, 4, len(mesh.Vertices))
	assert.Equal(t, vec.Vec3{1.0, 1.08866211, 1.15470054}, mesh.Max)
	assert.Equal(t, vec.Vec3{-1.0, -0.54433105, -0.57735027}, mesh.Min)
}

func TestCalculate(t *testing.T) {
	mesh, err := LoadOBJ("../tetrahedron.obj")
	assert.NoError(t, err)

	calculate(distanceSettings{
		width:  8,
		height: 8,
		depth:  8,
	}, *mesh)
}

func TestSameWind(t *testing.T) {
	tA := Triangle{0, 1, 2}
	tB := Triangle{1, 2, 3}
	tC := Triangle{3, 2, 1}

	r := sameWindingOrder(tA, tB, [2]uint32{2, 1})
	assert.False(t, r)

	r = sameWindingOrder(tA, tC, [2]uint32{2, 1})
	assert.True(t, r)
}

func TestTriangleList(t *testing.T) {
	mesh, err := LoadOBJ("../tetrahedron.obj")
	assert.NoError(t, err)

	width := 32
	height := 32
	depth := 32

	triangleLists := mesh.createTriangleLists(32, 32, 32)

	var pointScale, pointBias vec.Vec3

	pointScale[0] = (mesh.Max[0] - mesh.Min[0]) / float32(width-1)
	pointBias[0] = mesh.Min[0]

	pointScale[1] = (mesh.Max[1] - mesh.Min[1]) / float32(height-1)
	pointBias[1] = mesh.Min[1]

	pointScale[2] = (mesh.Max[2] - mesh.Min[2]) / float32(depth-1)
	pointBias[2] = mesh.Min[2]

	for z := 0; z < depth; z++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				p := vec.Add(vec.Mul(vec.Vec3{float32(x), float32(y), float32(z)}, pointScale), pointBias)
				d0 := mesh.distanceUsingList(p, uint(width), uint(height), uint(depth), uint(x), uint(y), uint(z), triangleLists)
				d1 := mesh.distance(p)

				assert.Equal(t, d0, d1)
			}
		}
	}
}
