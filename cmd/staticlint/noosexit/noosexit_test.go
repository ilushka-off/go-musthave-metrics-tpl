package noosexit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ilushka-off/go-musthave-metrics-tpl/cmd/staticlint/noosexit"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noosexit.Analyzer, "a")
}

func TestAnalyzer_SkipsGeneratedCode(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noosexit.Analyzer, "gen")
}
