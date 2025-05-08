package similarity

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"

	"github.com/agnivade/levenshtein"
	"github.com/atoscerebro/bms-analysis/pkg/ds"
	"gonum.org/v1/gonum/mat"
)

type Coordinate struct {
	X float64
	Y float64
}

type Comparable interface {
	Metric() string
}

func computeSimilarity(c1, c2 Comparable) float64 {
	a := c1.Metric()
	b := c2.Metric()

	if a == b {
		return 1.0
	}

	maxLen := math.Max(float64(len(a)), float64(len(b)))
	if maxLen == 0 {
		return 1.0
	}

	dist := float64(levenshtein.ComputeDistance(a, b))
	return 1.0 - dist/maxLen
}

func computeDistanceMatrix(c []Comparable) *mat.Dense {
	n := len(c)
	numCPU := runtime.NumCPU()
	chunkSize := (len(c) + numCPU - 1) / numCPU
	chunks := ds.SliceChunk(c, chunkSize)
	computed := map[string]bool{}
	totalComputed := 0
	dist := mat.NewDense(n, n, nil)
	mu := sync.Mutex{}

	var wg sync.WaitGroup
	for i := 0; i < len(chunks); i++ {
		wg.Add(1)
		go func(chunkI int) {
			defer wg.Done()
			currChunk := chunks[chunkI]
			baseI := chunkI * chunkSize
			for j := 0; j < len(currChunk); j++ {
				groupI := baseI + j
				for k := 0; k < n; k++ {
					key := fmt.Sprintf("%d-%d", groupI, k)
					mu.Lock()
					_, ok := computed[key]
					mu.Unlock()
					if ok {
						continue
					}
					sim := computeSimilarity(c[groupI], c[k])
					mu.Lock()
					dist.Set(groupI, k, 1.0-sim)
					dist.Set(k, groupI, 1.0-sim)
					computedKeys := []string{key, fmt.Sprintf("%d-%d", k, groupI)}
					for _, k := range computedKeys {
						computed[k] = true
					}
					mu.Unlock()
				}
				totalComputed++
				if totalComputed%(n/10) == 0 {
					log.Printf("%d of %d records compared...", totalComputed, n)
				}
			}
		}(i)
	}
	wg.Wait()
	return dist
}

func computeClassicalMDS(D *mat.Dense, dims int) (*mat.Dense, error) {
	n, _ := D.Dims()

	log.Printf("squaring distance matrix...")
	D2 := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			d := D.At(i, j)
			D2.Set(i, j, d*d)
		}
	}

	log.Printf("centering matrix...")
	I := mat.NewDense(n, n, nil)
	ones := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		I.Set(i, i, 1)
		for j := 0; j < n; j++ {
			ones.Set(i, j, 1.0/float64(n))
		}
	}

	C := mat.NewDense(n, n, nil)
	C.Sub(I, ones)

	log.Printf("computing scale...")
	// B = -0.5 * C * D2 * C
	tmp := mat.NewDense(n, n, nil)
	tmp.Mul(C, D2)
	B := mat.NewDense(n, n, nil)
	B.Mul(tmp, C)
	B.Scale(-0.5, B)

	log.Printf("performing eigen decomposition...")
	symB := mat.NewSymDense(n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			val := B.At(i, j)
			symB.SetSym(i, j, val)
		}
	}
	var eig mat.EigenSym
	if ok := eig.Factorize(symB, true); !ok {
		return nil, fmt.Errorf("eigen decomposition failed")
	}
	eigVals := eig.Values(nil)
	var eigVecs mat.Dense
	eig.VectorsTo(&eigVecs)

	log.Printf("calculating coordinates from top eigenvectors...")
	coords := mat.NewDense(n, dims, nil)
	for i := 0; i < dims; i++ {
		sqrtVal := math.Sqrt(eigVals[n-1-i])
		for j := 0; j < n; j++ {
			coords.Set(j, i, eigVecs.At(j, n-1-i)*sqrtVal)
		}
	}
	return coords, nil
}

func formatCoordinates(d *mat.Dense) []Coordinate {
	n, _ := d.Dims()
	coords := make([]Coordinate, n)
	for i := 0; i < n; i++ {
		coords[i] = Coordinate{
			X: d.At(i, 0),
			Y: d.At(i, 1),
		}
	}
	return coords
}

func Coordinates(c []Comparable) ([]Coordinate, error) {
	log.Printf("computing distance matrix...")
	d := computeDistanceMatrix(c)
	log.Printf("computing classical mds...")
	coords, err := computeClassicalMDS(d, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to compute mds: %s", err)
	}
	log.Printf("formatting coodinates...")
	return formatCoordinates(coords), nil
}
