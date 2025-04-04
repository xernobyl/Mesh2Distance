package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/chewxy/math32"
	"github.com/xernobyl/mesh2distance/src/vec"
)

// mirror modes
type convertionOptions uint16

const (
	convertionOptionsMirrorX              = 1 << 0
	convertionOptionsMirrorXIncludeCenter = 1 << 1
	convertionOptionsMirrorXNegative      = 1 << 2
	convertionOptionsMirrorY              = 1 << 3
	convertionOptionsMirrorYIncludeCenter = 1 << 4
	convertionOptionsMirrorYNegative      = 1 << 5
	convertionOptionsMirrorZ              = 1 << 6
	convertionOptionsMirrorZIncludeCenter = 1 << 7
	convertionOptionsMirrorZNegative      = 1 << 8

	convertionOptions16bits = 1 << 9 // 16 bits output, 8 bits otherwise
)

type distanceSettings struct {
	width             uint16
	height            uint16
	depth             uint16
	convertionOptions convertionOptions
}

func main() {
	// All the options that should be available to the user:
	// - Mirror modes, for each axis:
	//   - none
	//   - positive including center
	//   - positive excluding center
	//   - negative including center
	//   - negative excluding center
	// - Output type:
	//   - u8	(include bias and scale)
	//   - u16	(include bias and scale)
	// - Output resolution: eg 256x256x256

	// Output should be a binary blob, and a json file including:
	// - Bounding box
	// - Scale + bias for each axis
	// - Distance value scale + bias
	// - Mirror mode
	// - Output type
	// - Output resolution

	outputTypePtr := flag.Int("type", 8, "Output type, 8 or 16 bits")
	outputResolutionPtr := flag.String("res", "32x32x32", "Output resolution WIDTHxHEIGHTxDEPTH")
	mirrorModePtr := flag.String("mirrormode", "", "Mirroring mode for each axis... format to be determined")
	filePathPtr := flag.String("file", "bin", ".obj file path")
	formatPtr := flag.String("format", "bin", "output file format")
	checkFilePtr := flag.Bool("check", false, "Do some file checks before continuing, mostly for debugging")
	flag.Parse()

	distanceSettings := distanceSettings{}

	if *outputTypePtr != 8 && *outputTypePtr != 16 {
		fmt.Println("Output type must be 8 or 16")
		return
	}

	if *formatPtr != "bin" && *formatPtr != "dds" {
		fmt.Println("Output format must be \"bin\" or \"dds\"")
		return
	}

	if *outputTypePtr == 16 {
		distanceSettings.convertionOptions |= convertionOptions16bits
	}

	reMirror := regexp.MustCompile(`^(-?x?i?)(-?y?i?)(-?z?i?)$`)
	reSize0 := regexp.MustCompile(`^(\d{1,3})x(\d{1,3})x(\d{1,3})$`)
	reSize1 := regexp.MustCompile(`^(\d{1,3})$`)

	// Parse mirror modes (-xi, x, xi)
	if matches := reMirror.FindStringSubmatch(*mirrorModePtr); matches != nil {
		fmt.Printf("X mirror mode: \"%s\"\n", matches[1])
		fmt.Printf("Y mirror mode: \"%s\"\n", matches[2])
		fmt.Printf("Z mirror mode: \"%s\"\n", matches[3])
	} else {
		fmt.Println("Invalid mirror mode.")
		return
	}

	// Parse resolution
	if matches := reSize0.FindStringSubmatch(*outputResolutionPtr); matches != nil {
		w, _ := strconv.ParseUint(matches[1], 10, 16)
		h, _ := strconv.ParseUint(matches[2], 10, 16)
		d, _ := strconv.ParseUint(matches[3], 10, 16)

		if w <= 0 || h <= 0 || d <= 0 || w > 512 || h > 512 || d > 512 {
			fmt.Println("Output resolution must be between 1 and 512, inclusive")
			return
		}

		distanceSettings.width = uint16(w)
		distanceSettings.height = uint16(h)
		distanceSettings.depth = uint16(d)
	} else if matches := reSize1.FindStringSubmatch(*outputResolutionPtr); matches != nil {
		w, _ := strconv.ParseUint(matches[1], 10, 16)

		if w <= 0 || w > 512 {
			fmt.Println("Output resolution must be between 1 and 512, inclusive")
			return
		}

		distanceSettings.width = uint16(w)
	} else {
		fmt.Println("Invalid output resolution")
		return
	}

	fmt.Println("Loading 3D model...")

	mesh, err := LoadOBJ(*filePathPtr)
	if err != nil {
		fmt.Println("Error loading mesh:", err)
		return
	}

	// Do some weird checks on file, mostly for debugging
	if checkFilePtr != nil && *checkFilePtr {
		fmt.Println("Verifying mesh...")
		mesh.fixTriangles()
	}

	// Calculate other dimensions in case only one is given
	if distanceSettings.height == 0 && distanceSettings.depth == 0 {
		boxSize := vec.Sub(mesh.Max, mesh.Min)
		maxSide := vec.Max3(boxSize[0], boxSize[1], boxSize[2])
		var sw, sh, sd uint16

		if maxSide == boxSize[0] {
			sw = distanceSettings.width
			sh = uint16(math32.Ceil(boxSize[1] * float32(distanceSettings.width) / float32(boxSize[0])))
			sd = uint16(math32.Ceil(boxSize[2] * float32(distanceSettings.width) / float32(boxSize[0])))
		}

		if maxSide == boxSize[1] {
			sw = uint16(math32.Ceil(boxSize[0] * float32(distanceSettings.width) / float32(boxSize[1])))
			sh = distanceSettings.width
			sd = uint16(math32.Ceil(boxSize[2] * float32(distanceSettings.width) / float32(boxSize[1])))
		}

		if maxSide == boxSize[2] {
			sw = uint16(math32.Ceil(boxSize[0] * float32(distanceSettings.width) / float32(boxSize[2])))
			sh = uint16(math32.Ceil(boxSize[1] * float32(distanceSettings.width) / float32(boxSize[2])))
			sd = distanceSettings.width
		}

		distanceSettings.width = sw
		distanceSettings.height = sh
		distanceSettings.depth = sd
	}

	fmt.Printf("Output resolution: %d x %d x %d\n", distanceSettings.width, distanceSettings.height, distanceSettings.depth)

	data, minD, maxD := calculate(distanceSettings, *mesh)

	fmt.Println("Writing files...")

	ext := filepath.Ext(*filePathPtr)
	pathNoExt := strings.TrimSuffix(*filePathPtr, ext)

	if *formatPtr == "dds" {
		Save3DTextureAsDDS(
			pathNoExt+".dds",
			data,
			uint32(distanceSettings.width),
			uint32(distanceSettings.height),
			uint32(distanceSettings.depth),
			uint32(*outputTypePtr),
		)
	} else {
		err = os.WriteFile(pathNoExt+".bin", data, 0644)
		if err != nil {
			fmt.Println("Error saving file:", err)
			return
		}
	}

	jsonData, err := json.MarshalIndent(map[string]any{
		"distance_min":     minD,
		"distance_max":     maxD,
		"texture_width":    distanceSettings.width,
		"texture_height":   distanceSettings.height,
		"texture_depth":    distanceSettings.depth,
		"bounding_box_min": mesh.Min,
		"bounding_box_max": mesh.Max,
		"texture_data":     pathNoExt + "." + *formatPtr,
		"texture_format":   fmt.Sprintf("u%d", *outputTypePtr),
	}, "", "  ")
	if err != nil {
		panic(err)
	}

	// Write JSON to file
	err = os.WriteFile(pathNoExt+".json", jsonData, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("All done. Bye.")

	fmt.Printf("\nmodel(vec3(%f, %f, %f),\n\tvec3(%f, %f, %f),\n\t%f,\n\t%f);\n",
		mesh.Min[0], mesh.Min[1], mesh.Min[2],
		mesh.Max[0], mesh.Max[1], mesh.Max[2],
		minD,
		maxD)
}
