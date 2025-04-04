package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/chewxy/math32"

	"github.com/xernobyl/mesh2distance/src/vec"
)

type Triangle [3]uint32

type Mesh struct {
	Vertices  []vec.Vec3
	Triangles []Triangle
	Min       vec.Vec3 // Bounding box bottom corner
	Max       vec.Vec3 // Bounding box top corner
}

// Returns triangle bounding box
func getTriangleAABB(p0, p1, p2 vec.Vec3) (vec.Vec3, vec.Vec3) {
	var min vec.Vec3
	var max vec.Vec3

	for i := 0; i < 3; i++ {
		min[i] = vec.Min3(p0[i], p1[i], p2[i])
		max[i] = vec.Max3(p0[i], p1[i], p2[i])
	}

	return min, max
}

// Returns a list of triangles on each box square
func (m *Mesh) createTriangleLists(width, height, depth uint) [][]int {
	triangles := make([][]int, width*height*depth)

	for triangleIdx, triangle := range m.Triangles {
		v0 := m.Vertices[triangle[0]]
		v1 := m.Vertices[triangle[1]]
		v2 := m.Vertices[triangle[2]]

		min, max := getTriangleAABB(v0, v1, v2)

		var minIdx [3]uint
		var maxIdx [3]uint

		s := [3]uint{width, height, depth}

		for i := 0; i < 3; i++ {
			minIdx[i] = uint((min[i] - m.Min[i]) / (m.Max[i] - m.Min[i]) * float32(s[i]))
			maxIdx[i] = uint((max[i] - m.Min[i]) / (m.Max[i] - m.Min[i]) * float32(s[i]))
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

	model := &Mesh{
		Min: vec.Vec3{math32.Inf(1), math32.Inf(1), math32.Inf(1)},
		Max: vec.Vec3{math32.Inf(-1), math32.Inf(-1), math32.Inf(-1)},
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

			x, _ := strconv.ParseFloat(tokens[1], 32)
			y, _ := strconv.ParseFloat(tokens[2], 32)
			z, _ := strconv.ParseFloat(tokens[3], 32)
			model.Vertices = append(model.Vertices, vec.Vec3{float32(x), float32(y), float32(z)})

			model.Min[0] = min(model.Min[0], float32(x))
			model.Min[1] = min(model.Min[1], float32(y))
			model.Min[2] = min(model.Min[2], float32(z))

			model.Max[0] = max(model.Max[0], float32(x))
			model.Max[1] = max(model.Max[1], float32(y))
			model.Max[2] = max(model.Max[2], float32(z))

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
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return model, nil
}

/*
Signed distance from point p to triangle defined by a, b, and c.
*/
func distance(p, a, b, c vec.Vec3) float32 {
	ba := vec.Sub(b, a)
	pa := vec.Sub(p, a)
	cb := vec.Sub(c, b)
	pb := vec.Sub(p, b)
	ac := vec.Sub(a, c)
	pc := vec.Sub(p, c)
	n := vec.Cross(ba, ac)

	var sign float32
	if vec.Dot(n, pa) > 0.0 {
		sign = -1.0
	} else {
		sign = 1.0
	}

	if vec.Sign(vec.Dot(vec.Cross(ba, n), pa))+
		vec.Sign(vec.Dot(vec.Cross(cb, n), pa))+
		vec.Sign(vec.Dot(vec.Cross(ac, n), pa)) < 2.0 {
		return sign * math32.Sqrt(vec.Min3(
			vec.Dot2(vec.Sub(vec.Scale(ba, vec.Saturate(vec.Dot(ba, pa)/vec.Dot2(ba))), pa)),
			vec.Dot2(vec.Sub(vec.Scale(cb, vec.Saturate(vec.Dot(cb, pb)/vec.Dot2(cb))), pb)),
			vec.Dot2(vec.Sub(vec.Scale(ac, vec.Saturate(vec.Dot(ac, pc)/vec.Dot2(ac))), pc))))
	}

	return sign * math32.Sqrt((vec.Dot(n, pa) * vec.Dot(n, pa) / vec.Dot2(n)))
}

func isAdjacent(a, b Triangle) (bool, [2]uint32) {
	shared := [2]uint32{}
	count := 0

	for _, va := range a {
		for _, vb := range b {
			if va == vb {
				if count < 2 {
					shared[count] = va
				}
				count++
			}
		}
	}

	return count == 2, shared
}

func sameWindingOrder(triangleA, triangleB Triangle, shared [2]uint32) bool {
	for i, a := range triangleA {
		if a == shared[0] {
			if triangleA[(i+1)%3] == shared[1] {
				// other triangle should have shared[1] -> shared[0]

				for j, b := range triangleB {
					if b == shared[0] {
						return triangleB[(j+2)%3] == shared[1]
					}
				}
			} else {
				// other triangle should have shared[1] -> shared[0]

				for j, b := range triangleB {
					if b == shared[0] {
						return triangleB[(j+1)%3] == shared[1]
					}
				}
			}

		}
	}

	// should never be reached I think...
	return false
}

/*
Check if the triangles are pointing in a consistent direction
*/
func (mesh *Mesh) fixTriangle() bool {
	var wg sync.WaitGroup
	n := runtime.NumCPU()

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(a, b int) {
			for a, triangleA := range mesh.Triangles[a:b] {
				adjacentCount := 0

				for b, triangleB := range mesh.Triangles {
					if a == b {
						continue
					}

					adjacent, shared := isAdjacent(triangleA, triangleB)
					if !adjacent {
						continue
					}

					adjacentCount++

					sameWinding := sameWindingOrder(triangleA, triangleB, shared)
					if !sameWinding {

						fmt.Printf("Warning: Triangle %d (%d) is inverted! Check your 3d model.\n", b, a)

						/*t := triangleB[0]
						triangleB[0] = triangleB[1]
						triangleB[1] = t
						mesh.Triangles[b] = triangleB

						return false*/
					}
				}

				if adjacentCount == 0 {
					fmt.Println("Warning: Disconnected triangles! Check your 3D model.")
				}
			}

			wg.Done()
		}(i*(len(mesh.Triangles)/n), (i+1)*(len(mesh.Triangles)/n))
	}

	wg.Wait()

	return true
}

func (mesh *Mesh) fixTriangles() {
	for !mesh.fixTriangle() {
	}
}

/*
Signed distance from point p to closest point on mesh.
*/
func (mesh Mesh) distance(p vec.Vec3) float32 {
	minDistance := math32.Inf(1)

	for _, triangle := range mesh.Triangles {
		v0 := mesh.Vertices[triangle[0]]
		v1 := mesh.Vertices[triangle[1]]
		v2 := mesh.Vertices[triangle[2]]

		d := distance(p, v0, v1, v2)

		if d == 0 {
			return 0.0
		}

		if math32.Abs(d) < math32.Abs(minDistance) {
			minDistance = d
		}
	}

	return minDistance
}

/*
Signed distance from point p to closest point on mesh, using triangle lists to accelerate search
*/
func (m Mesh) distanceUsingList(p vec.Vec3, depth, height, width, x, y, z uint, triangleLists [][]int) float32 {
	visitedTriangles := map[int]struct{}{}
	foundTriangles := false
	layer := 0
	minDistance := math32.Inf(1)

	// if triangles are found in the layer then we won't find closer triangles on the next layers

	for !foundTriangles {
		for zz := vec.Max(0, int(z)-layer); zz <= vec.Min(int(z)+layer, int(depth-1)); zz++ {
			for yy := vec.Max(0, int(y)-layer); yy <= vec.Min(int(y)+layer, int(height-1)); yy++ {
				for xx := vec.Max(0, int(x)-layer); xx <= vec.Min(int(x)+layer, int(width-1)); xx++ {
					for _, triangleIdx := range triangleLists[xx+yy*int(width)+zz*int(width*height)] {
						// skip triangle if already visited
						_, ok := visitedTriangles[triangleIdx]
						if ok {
							continue
						}

						// set as visited
						visitedTriangles[triangleIdx] = struct{}{}
						foundTriangles = true

						triangle := m.Triangles[triangleIdx]
						v0 := m.Vertices[triangle[0]]
						v1 := m.Vertices[triangle[1]]
						v2 := m.Vertices[triangle[2]]

						d := distance(p, v0, v1, v2)

						// early exit, nothing smaller than 0.0 can be found
						if d == 0 {
							return 0.0
						}

						if math32.Abs(d) < math32.Abs(minDistance) {
							minDistance = d
						}
					}
				}
			}
		}

		// Go to next layer of the onion cube
		layer++
	}

	return minDistance
}

/*
Goes trough all points of 3D texture and calculates the signed distance to mesh.
*/
func calculate(settings distanceSettings, mesh Mesh) (outputData []byte, minD float32, maxD float32) {
	width := uint(settings.width)
	height := uint(settings.height)
	depth := uint(settings.depth)

	fmt.Println("Creating triangle lists...")
	triangleLists := mesh.createTriangleLists(width, height, depth)

	data := make([]float32, width*height*depth)

	// Minimum and maximum distance values (for normalization)
	minD = math32.Inf(1)
	maxD = math32.Inf(-1)

	var pointScale, pointBias vec.Vec3

	// Calculate scale and bias for each axis
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
		pointScale[0] = (mesh.Max[0] - mesh.Min[0]) / float32(width - 1)
		pointBias[0] = mesh.Min[0]
	}*/

	pointScale[0] = (mesh.Max[0] - mesh.Min[0]) / float32(width-1)
	pointBias[0] = mesh.Min[0]

	pointScale[1] = (mesh.Max[1] - mesh.Min[1]) / float32(height-1)
	pointBias[1] = mesh.Min[1]

	pointScale[2] = (mesh.Max[2] - mesh.Min[2]) / float32(depth-1)
	pointBias[2] = mesh.Min[2]

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Println("Calculating distance field...")

	for z := uint(0); z < depth; z++ {
		wg.Add(1)
		go func(z uint) {
			minDi := math32.Inf(1)
			maxDi := math32.Inf(-1)

			for y := uint(0); y < height; y++ {
				for x := uint(0); x < width; x++ {
					p := vec.Add(vec.Mul(vec.Vec3{float32(x), float32(y), float32(z)}, pointScale), pointBias)
					d := mesh.distanceUsingList(p, width, height, depth, x, y, z, triangleLists)
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

			fmt.Printf("finished layer %d out of %d\n", z, depth-1)
			wg.Done()
		}(z)
	}

	wg.Wait()
	fmt.Println("...All done.")

	/*
		// Find biggest smallest negative number adjacent to a positive number
		minDNextToPos := float32(0.0)

		for z := 1; z < int(depth)-2; z++ {
			for y := 1; y < int(height)-2; y++ {
				for x := 1; x < int(width)-2; x++ {

					d := data[x+y*int(width)+z*int(width)*int(height)]
					if d >= 0.0 {
						continue
					}

					t0 := data[(x-1)+y*int(width)+z*int(width)*int(height)]
					t1 := data[(x+1)+y*int(width)+z*int(width)*int(height)]
					t2 := data[x+(y-1)*int(width)+z*int(width)*int(height)]
					t3 := data[x+(y+1)*int(width)+z*int(width)*int(height)]
					t4 := data[x+y*int(width)+(z-1)*int(width)*int(height)]
					t5 := data[x+y*int(width)+(z+1)*int(width)*int(height)]

					if t0 > 0.0 || t1 > 0.0 || t2 > 0.0 || t3 > 0.0 || t4 > 0.0 || t5 > 0.0 {
						if d < minDNextToPos {
							minDNextToPos = d
						}
					}
				}
			}
		}

		fmt.Printf("minDNextToPos: %f\n", minDNextToPos)
	*/

	// Create buffer of correct type
	if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
		outputData = make([]byte, len(data)*2)
	} else {
		outputData = make([]byte, len(data))
	}

	printStep := len(data) / 100

	for i, v := range data {
		if i%printStep == 0 {
			fmt.Printf("\rConverting data:\t%d%%", i/printStep)
		}

		negative := v < 0.0
		v = (v - minD) / (maxD - minD) // normalize to [0, 1]

		// Convert to 8 or 16 bits... Rounding up or down depending on the sign of the distance value
		if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
			var t uint16
			if negative {
				t = uint16(vec.Max(0.0, vec.Min(math32.Floor(v*65535.0), 65535.0)))
			} else {
				t = uint16(vec.Max(0.0, vec.Min(math32.Ceil(v*65535.0), 65535.0)))
			}

			// Convert to little endian
			outputData[i*2] = byte(t & 0xFF)
			outputData[i*2+1] = byte((t >> 8) & 0xFF)
		} else {
			if negative {
				outputData[i] = uint8(vec.Max(0.0, vec.Min(math32.Floor(v*255), 255.0)))
			} else {
				outputData[i] = uint8(vec.Max(0.0, vec.Min(math32.Ceil(v*255), 255.0)))
			}
		}
	}

	fmt.Println("\rConverting data:\t100%")

	return outputData, minD, maxD
}
