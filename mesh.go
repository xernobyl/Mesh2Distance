package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chewxy/math32"
)

type Triangle [3]uint32

type Mesh struct {
	Vertices  []Vec3
	Triangles []Triangle
	Min       Vec3    // Bounding box bottom corner
	Max       Vec3    // Bounding box top corner
	Center    Vec3    // Bounding sphere center
	Radius    float32 // Bounding sphere radius
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
		Min: Vec3{math32.Inf(1), math32.Inf(1), math32.Inf(1)},
		Max: Vec3{math32.Inf(-1), math32.Inf(-1), math32.Inf(-1)},
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
			model.Vertices = append(model.Vertices, Vec3{float32(x), float32(y), float32(z)})

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
func distance(p, a, b, c Vec3) float32 {
	ba := Sub(b, a)
	pa := Sub(p, a)
	cb := Sub(c, b)
	pb := Sub(p, b)
	ac := Sub(a, c)
	pc := Sub(p, c)
	n := Cross(ba, ac)

	var sign float32
	if t := Dot(n, pa); t >= 0.0 {
		sign = 1.0
	} else {
		sign = -1.0
	}

	if Sign(Dot(Cross(ba, n), pa))+
		Sign(Dot(Cross(cb, n), pa))+
		Sign(Dot(Cross(ac, n), pa)) < 2.0 {
		return sign * math32.Sqrt(Min3(
			Dot2(Sub(Scale(ba, Saturate(Dot(ba, pa)/Dot2(ba))), pa)),
			Dot2(Sub(Scale(cb, Saturate(Dot(cb, pb)/Dot2(cb))), pb)),
			Dot2(Sub(Scale(ac, Saturate(Dot(ac, pc)/Dot2(ac))), pc))))
	}

	return sign * math32.Sqrt((Dot(n, pa) * Dot(n, pa) / Dot2(n)))
}

/*
Signed distance from point p to closest point on mesh.
*/
func (mesh Mesh) distance(p Vec3) float32 {
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
Goes trough all points of 3D texture and calculates the signed distance to mesh.
*/
func calculate(settings distanceSettings, mesh Mesh) (outputData []byte, minD float32, maxD float32) {
	width := uint(settings.width)
	height := uint(settings.height)
	depth := uint(settings.depth)

	data := make([]float32, width*height*depth)
	i := 0

	// Minimum and maximum distance values (for normalization)
	minD = math32.Inf(1)
	maxD = math32.Inf(-1)

	var pointScale, pointBias Vec3

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

	printStep := int(width) * int(height) * int(depth) / 100

	for z := uint(0); z < depth; z++ {
		for y := uint(0); y < height; y++ {
			for x := uint(0); x < width; x++ {
				if i%printStep == 0 {
					fmt.Printf("\rCalculating distance values:\t%d%%", i/printStep)
				}

				p := Add(Mul(Vec3{float32(x), float32(y), float32(z)}, pointScale), pointBias)
				d := mesh.distance(p)
				data[i] = d
				i += 1

				if d < minD {
					minD = d
				}

				if d > maxD {
					maxD = d
				}
			}
		}
	}

	fmt.Println("\rCalculating distance values:\t100%")

	// Create buffer of correct type
	if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
		outputData = make([]byte, len(data)*2)
	} else {
		outputData = make([]byte, len(data))
	}

	printStep = len(data) / 100

	for i, v := range data {
		if i%printStep == 0 {
			fmt.Printf("\rConverting data:\t%d%%", i/printStep)
		}

		negative := v < 0.0

		if settings.convertionOptions&convertionOptionsSquare == convertionOptionsSquare {
			zero := -minD / (maxD - minD) // zero point in [0, 1]

			// maybe I should store the zero explicitly in the data
			// however the probability of a distance being exactly zero is very low
			// so let's just keep this here for now

			// f(minD) = 0.0
			// f(maxD) = 1.0

			// f⁻¹(0.0) = minD
			// f⁻¹(zero) = 0.0
			// f⁻¹(1.0) = maxD

			if v == 0.0 {
				v = zero
			} else if negative {
				v = zero - math32.Sqrt(-v)*zero/math32.Sqrt(-minD)
			} else {
				v = zero + math32.Sqrt(v*zero*zero-2.0*v*zero+v)/math32.Sqrt(maxD)
			}
		} else {
			// linear, normalize to [0, 1]
			v = (v - minD) / (maxD - minD)
		}

		// Convert to 8 or 16 bits... Rounding up or down depending on the sign of the distance value
		if settings.convertionOptions&convertionOptions16bits == convertionOptions16bits {
			var t uint16
			if negative {
				t = uint16(math32.Floor(v * 65535))
			} else {
				t = uint16(math32.Ceil(v * 65535))
			}

			// Convert to little endian
			outputData[i*2] = byte(t & 0xFF)
			outputData[i*2+1] = byte((t >> 8) & 0xFF)
		} else {
			if negative {
				outputData[i] = uint8(math32.Floor(v * 255))
			} else {
				outputData[i] = uint8(math32.Ceil(v * 255))
			}
		}
	}

	fmt.Println("\rConverting data:\t100%")

	return outputData, minD, maxD
}
