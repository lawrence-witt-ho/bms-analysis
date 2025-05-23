package main

import (
	"github.com/gzuidhof/tygo/tygo"
)

func main() {
	config := &tygo.Config{
		Packages: []*tygo.PackageConfig{
			{
				Path:       "github.com/atoscerebro/bms-analysis/internal/kibana",
				OutputPath: "internal/client/src/models/kibana.ts",
			},
		},
	}
	gen := tygo.New(config)
	if err := gen.Generate(); err != nil {
		panic(err)
	}
}
