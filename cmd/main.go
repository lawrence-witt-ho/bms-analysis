package main

import (
	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/atoscerebro/bms-analysis/internal/kibana"
)

func main() {
	cf, err := config.Load()
	if err != nil {
		panic(err)
	}
	kc := kibana.NewKibanaClient(cf)
	err = kc.AnalyseErrorKeywords()
	if err != nil {
		panic(err)
	}
}
