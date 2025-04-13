package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"math"

	"github.com/xernobyl/mesh2distance/src/vec"
)

type Triangle [3]uint32

type edgeKey struct {
	A, B uint32
}

type Mesh struct {
	Vertices  []vec.Vec3
	Triangles []Triangle
	Min       vec.Vec3 // Bounding box bottom corner
	Max       vec.Vec3 // Bounding box top corner
}

var posBigfloat64 = math.Nextafter(math.Inf(1.0), -1.0)
var negBigfloat64 = math.Nextafter(math.Inf(-1.0), 1.0)
var posSmallfloat64 = math.Nextafter(0.0, 1.0)
var negSmallfloat64 = math.Nextafter(0.0, -1.0)

// Calculate other dimensions in case only one is given, using cubic
// texels, because there's no clear advantage to using square textures,
// and add 0.5 texels on each side of the mesh to avoid artifacts.
func calculateGridSize(meshMin, meshMax vec.Vec3, resolution int) (w, h, d int, gridMin, gridMax vec.Vec3) {
	var gridSize vec.Vec3
	meshSize := vec.Sub(meshMax, meshMin)

	biggestSide := vec.Max3(meshSize[0], meshSize[1], meshSize[2])
	density := float64(resolution) / biggestSide
	iDensity := biggestSide / float64(resolution)
	bigGridSize := biggestSide * float64(resolution-1) / float64(resolution-2)
	texel := bigGridSize - biggestSide

	if biggestSide == meshSize[0] {
		// X is the biggest side
		w = resolution
		h = int(math.Ceil((meshSize[1] + texel) * density))
		d = int(math.Ceil((meshSize[2] + texel) * density))

		gridSize[0] = bigGridSize
		gridSize[1] = float64(h) * iDensity
		gridSize[2] = float64(d) * iDensity
	} else if biggestSide == meshSize[1] {
		// Y is the biggest side
		w = int(math.Ceil((meshSize[0] + texel) * density))
		h = resolution
		d = int(math.Ceil((meshSize[2] + texel) * density))

		gridSize[0] = float64(w) * iDensity
		gridSize[1] = bigGridSize
		gridSize[2] = float64(d) * iDensity
	} else {
		// Z is the biggest side
		w = int(math.Ceil((meshSize[0] + texel) * density))
		h = int(math.Ceil((meshSize[1] + texel) * density))
		d = resolution

		gridSize[0] = float64(w) * iDensity
		gridSize[1] = float64(h) * iDensity
		gridSize[2] = bigGridSize
	}

	diff := vec.Scale(vec.Sub(gridSize, meshSize), 0.5)
	gridMin = vec.Sub(meshMin, diff)
	gridMax = vec.Add(meshMax, diff)

	return w, h, d, gridMin, gridMax
}

// Returns triangle bounding box
func getTriangleAABB(v0, v1, v2 vec.Vec3) (vec.Vec3, vec.Vec3) {
	var min vec.Vec3
	var max vec.Vec3

	for i := range 3 {
		min[i] = vec.Min3(v0[i], v1[i], v2[i])
		max[i] = vec.Max3(v0[i], v1[i], v2[i])
	}

	return min, max
}

// Returns a list of triangles on each box square
func (m *Mesh) createTriangleLists(width, height, depth int, gridMin, gridMax vec.Vec3) [][]int {
	triangles := make([][]int, width*height*depth)

	for triangleIdx, triangle := range m.Triangles {
		v0 := m.Vertices[triangle[0]]
		v1 := m.Vertices[triangle[1]]
		v2 := m.Vertices[triangle[2]]

		tMin, tMax := getTriangleAABB(v0, v1, v2)
		s := [3]int{width - 1, height - 1, depth - 1}
		var minIdx [3]int
		var maxIdx [3]int

		// calculate the min and max indices of the triangle in the grid
		for i := range 3 {
			minIdx[i] = int(math.Floor((tMin[i] - gridMin[i]) / (gridMax[i] - gridMin[i]) * float64(s[i])))
			maxIdx[i] = int(math.Ceil((tMax[i] - gridMin[i]) / (gridMax[i] - gridMin[i]) * float64(s[i])))
		}

		for z := vec.Max(0, minIdx[2]); z <= vec.Min(depth-1, maxIdx[2]); z++ {
			for y := vec.Max(0, minIdx[1]); y <= vec.Min(height-1, maxIdx[1]); y++ {
				for x := vec.Max(0, minIdx[0]); x <= vec.Min(width-1, maxIdx[0]); x++ {
					triangles[x+y*width+z*width*height] = append(triangles[x+y*width+z*width*height], triangleIdx)
				}
			}
		}
	}

	return triangles
}

// LoadOBJ loads a mesh from an OBJ file.
// It parses the vertices and triangular faces, and calculates the bounding box.
func LoadOBJ(filepath string) (*Mesh, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	edgeCount := make(map[edgeKey]int)

	verts := make(map[vec.Vec3][]int)

	model := &Mesh{
		Min: vec.Vec3{posBigfloat64, posBigfloat64, posBigfloat64},
		Max: vec.Vec3{negBigfloat64, negBigfloat64, negBigfloat64},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		switch tokens[0] {
		case "v":
			if len(tokens) != 4 {
				return nil, fmt.Errorf("unexpected number of vertices: %s", line)
			}

			x, _ := strconv.ParseFloat(tokens[1], 64)
			y, _ := strconv.ParseFloat(tokens[2], 64)
			z, _ := strconv.ParseFloat(tokens[3], 64)
			vertex := vec.Vec3{float64(x), float64(y), float64(z)}
			verts[vertex] = append(verts[vertex], len(model.Vertices))
			model.Vertices = append(model.Vertices, vertex)

			model.Min[0] = min(model.Min[0], float64(x))
			model.Min[1] = min(model.Min[1], float64(y))
			model.Min[2] = min(model.Min[2], float64(z))

			model.Max[0] = max(model.Max[0], float64(x))
			model.Max[1] = max(model.Max[1], float64(y))
			model.Max[2] = max(model.Max[2], float64(z))

			if len(verts[vertex]) > 1 {
				fmt.Println("Warning: mesh has duplicated vertices.")
			}

		case "f":
			if len(tokens) != 4 {
				return nil, fmt.Errorf("only triangular faces supported: %s", line)
			}

			v0, _ := strconv.Atoi(strings.Split(tokens[1], "/")[0])
			v1, _ := strconv.Atoi(strings.Split(tokens[2], "/")[0])
			v2, _ := strconv.Atoi(strings.Split(tokens[3], "/")[0])

			var triangle Triangle
			triangle = Triangle{uint32(v0 - 1), uint32(v1 - 1), uint32(v2 - 1)}
			model.Triangles = append(model.Triangles, triangle)

			edges := [3][2]uint32{
				{triangle[0], triangle[1]},
				{triangle[1], triangle[2]},
				{triangle[2], triangle[0]},
			}

			for _, e := range edges {
				// Sort the edge to make it undirected
				a, b := e[0], e[1]
				if a > b {
					a, b = b, a
				}
				key := edgeKey{A: a, B: b}
				edgeCount[key]++
			}
		}
	}

	for _, count := range edgeCount {
		if count != 2 {
			return nil, fmt.Errorf("mesh is not watertight")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	fmt.Printf("%d triangles\n", len(model.Triangles))

	return model, nil
}

/*
Signed distance from point p to triangle defined by a, b, and c.
*/
func distance(p, a, b, c vec.Vec3) float64 {
	ba := vec.Sub(b, a)
	pa := vec.Sub(p, a)
	cb := vec.Sub(c, b)
	pb := vec.Sub(p, b)
	ac := vec.Sub(a, c)
	pc := vec.Sub(p, c)
	n := vec.Cross(ba, ac)

	var sign float64
	t := vec.Dot(n, vec.Sub(p, vec.Scale(vec.Add(vec.Add(b, c), a), 1.0/3.0)))

	if t == 0.0 {
		return posBigfloat64
	}

	if t > 0.0 {
		sign = -1.0
	} else {
		sign = 1.0
	}

	if vec.Sign(vec.Dot(vec.Cross(ba, n), pa))+
		vec.Sign(vec.Dot(vec.Cross(cb, n), pa))+
		vec.Sign(vec.Dot(vec.Cross(ac, n), pa)) < 2.0 {
		return math.Copysign(math.Sqrt(vec.Min3(
			vec.Dot2(vec.Sub(vec.Scale(ba, vec.Saturate(vec.Dot(ba, pa)/vec.Dot2(ba))), pa)),
			vec.Dot2(vec.Sub(vec.Scale(cb, vec.Saturate(vec.Dot(cb, pb)/vec.Dot2(cb))), pb)),
			vec.Dot2(vec.Sub(vec.Scale(ac, vec.Saturate(vec.Dot(ac, pc)/vec.Dot2(ac))), pc)))), sign)
	}

	return math.Copysign(math.Sqrt((vec.Dot(n, pa) * vec.Dot(n, pa) / vec.Dot2(n))), sign)
}

/*
Signed distance from point p to closest point on mesh, using triangle lists to accelerate search
*/
func (m Mesh) distanceUsingList(p vec.Vec3, width, height, depth, ix, iy, iz int, triangleLists [][]int) float64 {
	visitedTriangles := map[int]struct{}{}
	foundTriangles := false
	layer := 0
	minDistance := posBigfloat64
	checkedTriangles := 0

	processTriangle := func(triangleIdx int) {
		triangle := m.Triangles[triangleIdx]
		v0 := m.Vertices[triangle[0]]
		v1 := m.Vertices[triangle[1]]
		v2 := m.Vertices[triangle[2]]
		d := distance(p, v0, v1, v2)

		if math.Abs(d) < math.Abs(minDistance) {
			minDistance = d
		}
	}

	if triangleLists == nil {
		for i := range m.Triangles {
			processTriangle(i)
			if minDistance == 0.0 {
				return 0.0
			}
		}
	} else {
		// if triangles are found in the layer then we won't find closer triangles on the next layers
		for !foundTriangles {
			for zz := vec.Max(0, int(iz)-layer); zz <= vec.Min(int(iz)+layer, int(depth-1)); zz++ {
				for yy := vec.Max(0, int(iy)-layer); yy <= vec.Min(int(iy)+layer, int(height-1)); yy++ {
					for xx := vec.Max(0, int(ix)-layer); xx <= vec.Min(int(ix)+layer, int(width-1)); xx++ {
						for _, triangleIdx := range triangleLists[xx+yy*int(width)+zz*int(width*height)] {
							// skip triangle if already visited
							_, ok := visitedTriangles[triangleIdx]
							if ok {
								continue
							}

							checkedTriangles++

							// set as visited
							visitedTriangles[triangleIdx] = struct{}{}
							foundTriangles = true

							processTriangle(triangleIdx)
							if minDistance == 0.0 {
								return 0.0
							}
						}
					}
				}
			}

			// Go to next layer of the cubic onion
			layer++
		}
	}

	return minDistance
}

/*
Goes trough all points of 3D texture and calculates the signed distance to mesh.
*/
func calculate(settings distanceSettings, mesh Mesh, gridMin, gridMax vec.Vec3) (outputData []byte, minD float64, maxD float64) {
	width := int(settings.width)
	height := int(settings.height)
	depth := int(settings.depth)

	fmt.Println("Creating triangle lists...")
	triangleLists := mesh.createTriangleLists(width, height, depth, gridMin, gridMax)

	data := make([]float64, width*height*depth)

	// Minimum and maximum distance values (for normalization)
	minD = negSmallfloat64
	maxD = posSmallfloat64

	var pointScale, pointBias vec.Vec3

	// Calculate scale and bias for each axis
	// TODO: mirror modes!
	/*if settings.convertionOptions&convertionOptionsMirrorX == convertionOptionsMirrorX {
		if settings.convertionOptions&convertionOptionsMirrorXIncludeCenter == convertionOptionsMirrorXIncludeCenter {
			// ...
		} else {
			// ...
		}

		if settings.convertionOptions&convertionOptionsMirrorXNegative == convertionOptionsMirrorXNegative {
			pointScale[0] = -pointScale[0]
			pointBias[0] = -pointBias[0]
		}
	} else {
		pointScale[0] = (mesh.Max[0] - mesh.Min[0]) / float64(width - 1)
		pointBias[0] = mesh.Min[0]
	}*/

	pointScale[0] = (gridMax[0] - gridMin[0]) / float64(width-1)
	pointBias[0] = gridMin[0]

	pointScale[1] = (gridMax[1] - gridMin[1]) / float64(height-1)
	pointBias[1] = gridMin[1]

	pointScale[2] = (gridMax[2] - gridMin[2]) / float64(depth-1)
	pointBias[2] = gridMin[2]

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Println("Calculating distance field:")

	maxSize := 0.5 * vec.Length(vec.Sub(gridMax, gridMin))

	progress := int32(0)
	progressStep := int32(width * height * depth / 100)

	for z := range depth {
		wg.Add(1)
		go func(z int) {
			minDi := negSmallfloat64
			maxDi := posSmallfloat64

			for y := range height {
				previousValue := posBigfloat64

				for x := range width {
					if progress%progressStep == 0 {
						fmt.Printf("\r%d%%", progress/progressStep)
					}
					atomic.AddInt32(&progress, 1)

					p := vec.Add(vec.Mul(vec.Vec3{float64(x), float64(y), float64(z)}, pointScale), pointBias)
					d := mesh.distanceUsingList(p, width, height, depth, x, y, z, triangleLists)

					// HACK to fix sign goes here

					// Ignore first value
					if previousValue != posBigfloat64 && d != 0.0 && math.Abs(previousValue-d) > math.Sqrt(2.0)*pointScale[0] && previousValue*d < 0.0 {
						fmt.Println("Inverting sign.")
						d = math.Copysign(d, previousValue)
					}

					previousValue = d
					data[x+y*width+z*width*height] = d

					if d < minDi {
						minDi = d
					}

					if d > maxDi {
						maxDi = d
					}
				}
			}

			mu.Lock()
			if minDi < minD {
				minD = minDi
			}
			if maxDi > maxD {
				maxD = maxDi
			}
			mu.Unlock()

			wg.Done()
		}(z)
	}

	wg.Wait()

	// Clamp the min and max distance values to the size of the grid
	minD = vec.Max(minD, -maxSize)
	maxD = vec.Min(maxD, maxSize)

	fmt.Println("\r100%")

	// Create buffer of correct type
	if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
		outputData = make([]byte, len(data)*2)
	} else {
		outputData = make([]byte, len(data))
	}

	fmt.Println("Converting data:")
	printStep := len(data) / 100

	for i, v := range data {
		if i%printStep == 0 {
			fmt.Printf("\r%d%%", i/printStep)
		}

		negative := v < 0.0
		v = (v - minD) / (maxD - minD) // normalize to [0, 1]

		// Convert to 8 or 16 bits... Rounding up or down depending on the sign of the distance value
		if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
			var t uint16
			if negative {
				t = uint16(vec.Max(0.0, vec.Min(math.Floor(v*65535.0), 65535.0)))
			} else {
				t = uint16(vec.Max(0.0, vec.Min(math.Ceil(v*65535.0), 65535.0)))
			}

			// Convert to little endian
			outputData[i*2] = byte(t & 0xFF)
			outputData[i*2+1] = byte((t >> 8) & 0xFF)
		} else {
			if negative {
				outputData[i] = uint8(vec.Max(0.0, vec.Min(math.Floor(v*255), 255.0)))
			} else {
				outputData[i] = uint8(vec.Max(0.0, vec.Min(math.Ceil(v*255), 255.0)))
			}
		}
	}

	fmt.Println("\r100%")

	return outputData, minD, maxD
}
