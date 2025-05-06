package similarity

import (
	"fmt"
	"log"
	"math"

	"github.com/adrg/strutil"
	metrics "github.com/adrg/strutil/metrics"
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
	maxLen := math.Max(float64(len(c1.Metric())), float64(len(c2.Metric())))
	if maxLen == 0 {
		return 1.0
	}
	dist := float64(strutil.Similarity(c1.Metric(), c2.Metric(), metrics.NewLevenshtein()))
	return 1.0 - dist/maxLen
}

func computeDistanceMatrix(c []Comparable) *mat.Dense {
	n := len(c)
	dist := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		if i%10 == 0 {
			log.Printf("%d of %d records compared", i, n)
		}
		for j := 0; j < n; j++ {
			sim := computeSimilarity(c[i], c[j])
			dist.Set(i, j, 1.0-sim)
		}
	}
	return dist
}

func computeClassicalMDS(D *mat.Dense, dims int) (*mat.Dense, error) {
	n, _ := D.Dims()

	// Step 1: Square the distance matrix
	D2 := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			d := D.At(i, j)
			D2.Set(i, j, d*d)
		}
	}

	// Step 2: Centering matrix
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

	// Step 3: Compute B = -0.5 * C * D2 * C
	tmp := mat.NewDense(n, n, nil)
	tmp.Mul(C, D2)
	B := mat.NewDense(n, n, nil)
	B.Mul(tmp, C)
	B.Scale(-0.5, B)

	// Step 4: Eigen-decomposition
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

	// Step 5: Use top eigenvectors for coordinates
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
	d := computeDistanceMatrix(c)
	coords, err := computeClassicalMDS(d, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to compute mds: %s", err)
	}
	return formatCoordinates(coords), nil
}
