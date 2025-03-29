package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistance(t *testing.T) {
	a := Vec3{0.0, 1.0, -1.0}
	b := Vec3{0.0, -1.0, -1.0}
	c := Vec3{0.0, 0.0, 1.0}

	p := Vec3{-1.0, 0, 0}
	d0 := distance(p, a, b, c)
	d1 := distance(p, b, a, c)

	assert.Equal(t, -d0, d1) // distance should be symmetric
	assert.Equal(t, float32(-1.095445), d0)

	p = Vec3{0.0, 1.0, -1.0}
	d0 = distance(p, b, a, c)
	assert.Equal(t, float32(0.0), d0)

	p = Vec3{0.0, 3.0, -1.0}
	d0 = distance(p, b, a, c)
	assert.Equal(t, float32(2.0), d0)
}

func TestLoadOBJ(t *testing.T) {
	mesh, err := LoadOBJ("tetrahedron.obj")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(mesh.Triangles))
	assert.Equal(t, 4, len(mesh.Vertices))
	assert.Equal(t, Vec3{1.0, 1.08866211, 1.15470054}, mesh.Max)
	assert.Equal(t, Vec3{-1.0, -0.54433105, -0.57735027}, mesh.Min)
}

func TestCalculate(t *testing.T) {
	mesh, err := LoadOBJ("tetrahedron.obj")
	assert.NoError(t, err)

	calculate(distanceSettings{
		width:  8,
		height: 8,
		depth:  8,
	}, *mesh)
}
