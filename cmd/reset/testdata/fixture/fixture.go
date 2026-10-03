// Package fixture is a manual test target for cmd/reset: it is not built
// by "go build ./..." or "go test ./..." (the "testdata" directory is
// excluded by convention), but can be regenerated and exercised directly,
// e.g.:
//
//	go run ./cmd/reset ./cmd/reset/testdata/fixture
//	go test ./cmd/reset/testdata/fixture/...
package fixture

// generate:reset
type ResetableStruct struct {
	i     int
	str   string
	strP  *string
	s     []int
	m     map[string]string
	child *ResetableStruct
}

// generate:reset
type WithEmbedded struct {
	ResetableStruct
	Extra bool
}
