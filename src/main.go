package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// mirror modes
type convertionOptions uint16

const resLimit = 4096
const sizeLimit = 16 * 1024 * 1024 // 16MB

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
	// - Output resolution biggest dimension

	// Output should be a binary blob, and a json file including:
	// - Bounding box
	// - Scale + bias for each axis
	// - Distance value scale + bias
	// - Mirror mode
	// - Output type
	// - Output resolution

	outputTypePtr := flag.Int("type", 8, "Output type, 8 or 16 bits")
	outputResolutionPtr := flag.Int("res", 32, "Output resolution biggest side")
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

	if *outputResolutionPtr < 16 && *outputResolutionPtr > resLimit {
		fmt.Printf("Output resolution must be between 16 and %d.\n", resLimit)
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

	// Parse mirror modes (-xi, x, xi)
	if matches := reMirror.FindStringSubmatch(*mirrorModePtr); matches != nil {
		fmt.Printf("X mirror mode: \"%s\"\n", matches[1])
		fmt.Printf("Y mirror mode: \"%s\"\n", matches[2])
		fmt.Printf("Z mirror mode: \"%s\"\n", matches[3])

		switch matches[1] {
		case "x":
			distanceSettings.convertionOptions |= convertionOptionsMirrorX
		case "-x":
			distanceSettings.convertionOptions |= convertionOptionsMirrorXNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorX
		case "xi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorX
			distanceSettings.convertionOptions |= convertionOptionsMirrorXIncludeCenter
		case "-xi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorXNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorX
			distanceSettings.convertionOptions |= convertionOptionsMirrorXIncludeCenter
		}

		switch matches[2] {
		case "y":
			distanceSettings.convertionOptions |= convertionOptionsMirrorY
		case "-y":
			distanceSettings.convertionOptions |= convertionOptionsMirrorYNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorY
		case "yi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorY
			distanceSettings.convertionOptions |= convertionOptionsMirrorYIncludeCenter
		case "-yi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorYNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorY
			distanceSettings.convertionOptions |= convertionOptionsMirrorYIncludeCenter
		}

		switch matches[3] {
		case "z":
			distanceSettings.convertionOptions |= convertionOptionsMirrorZ
		case "-z":
			distanceSettings.convertionOptions |= convertionOptionsMirrorZNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorZ
		case "zi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorZ
			distanceSettings.convertionOptions |= convertionOptionsMirrorZIncludeCenter
		case "-zi":
			distanceSettings.convertionOptions |= convertionOptionsMirrorZNegative
			distanceSettings.convertionOptions |= convertionOptionsMirrorZ
			distanceSettings.convertionOptions |= convertionOptionsMirrorZIncludeCenter
		}
	} else {
		fmt.Println("Invalid mirror mode.")
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

	w, h, d, gridMin, gridMax := calculateGridSize(mesh.Min, mesh.Max, *outputResolutionPtr)
	fmt.Printf("Output resolution: %d x %d x %d\n", w, h, d)

	if (w * h * d) > sizeLimit {
		fmt.Printf("Output size is too big, maximum allowed is %d bytes.\n", sizeLimit)
		return
	}

	distanceSettings.width = uint16(w)
	distanceSettings.height = uint16(h)
	distanceSettings.depth = uint16(d)

	data, minD, maxD := calculate(distanceSettings, *mesh, gridMin, gridMax)

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
		"distance_min":          minD,
		"distance_max":          maxD,
		"texture_width":         distanceSettings.width,
		"texture_height":        distanceSettings.height,
		"texture_depth":         distanceSettings.depth,
		"mesh_bounding_box_min": mesh.Min,
		"mesh_bounding_box_max": mesh.Max,
		"grid_bounding_box_min": gridMin,
		"grid_bounding_box_max": gridMax,
		"texture_data":          pathNoExt + "." + *formatPtr,
		"texture_format":        fmt.Sprintf("u%d", *outputTypePtr),
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
		gridMin[0], gridMin[1], gridMin[2],
		gridMax[0], gridMax[1], gridMax[2],
		minD,
		maxD)
}
