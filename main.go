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

	convertionOptions16bits = 1 << 9  // 16 bits output, 8 bits otherwise
	convertionOptionsSquare = 1 << 10 // Output quadratic, linear otherwise
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
	// - Bounding sphere
	// - Scale + bias for each axis
	// - Distance value scale + bias
	// - Mirror mode
	// - Output type
	// - Output resolution
	// - Quadratic or linear output

	outputTypePtr := flag.Int("outputtype", 8, "Output type, 8 or 16 bits")
	outputResolutionPtr := flag.String("outputresolution", "32x32x32", "Output resolution WIDTHxHEIGHTxDEPTH")
	//mirrorModePtr := flag.String("mirrormode", "", "Mirroring mode for each axis... format to be determined")
	filePathPtr := flag.String("file", "", ".obj file path")
	flag.Parse()

	distanceSettings := distanceSettings{}

	if *outputTypePtr != 8 && *outputTypePtr != 16 {
		fmt.Println("Output type must be 8 or 16")
		return
	}

	if *outputTypePtr == 16 {
		distanceSettings.convertionOptions |= convertionOptions16bits
	}

	re := regexp.MustCompile(`(\d{1,3})x(\d{1,3})x(\d{1,3})`)

	if matches := re.FindStringSubmatch(*outputResolutionPtr); matches == nil {
		fmt.Println("Invalid output resolution")
		return
	} else {
		w, _ := strconv.ParseUint(matches[1], 10, 16)
		distanceSettings.width = uint16(w)
		h, _ := strconv.ParseUint(matches[2], 10, 16)
		distanceSettings.height = uint16(h)
		d, _ := strconv.ParseUint(matches[3], 10, 16)
		distanceSettings.depth = uint16(d)

		if w <= 0 || h <= 0 || d <= 0 || w > 256 || h > 256 || d > 256 {
			fmt.Println("Output resolution must be between 1 and 256, inclusive")
			return
		}

		fmt.Printf("Output resolution: %d x %d x %d\n", w, h, d)
	}

	mesh, err := LoadOBJ(*filePathPtr)
	if err != nil {
		fmt.Println("Error loading mesh:", err)
		return
	}

	data, minD, maxD := calculate(distanceSettings, *mesh)

	fmt.Println("Writing files...")

	ext := filepath.Ext(*filePathPtr)
	pathNoExt := strings.TrimSuffix(*filePathPtr, ext)

	err = os.WriteFile(pathNoExt+".bin", data, 0644)
	if err != nil {
		fmt.Println("Error saving file:", err)
		return
	}

	jsonData, err := json.MarshalIndent(map[string]any{
		"min": minD,
		"max": maxD,
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
}
