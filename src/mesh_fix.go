package main

import (
	"fmt"
	"runtime"
	"sync"
)

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
