package agent

import "testing"

func TestCollectRunTimeGauges(t *testing.T) {
	gauges := CollectRunTimeGauges()

	want := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range want {
		if _, ok := gauges[name]; !ok {
			t.Errorf("missing gauge %q in collected metrics", name)
		}
	}

	if len(gauges) != len(want) {
		t.Errorf("got %d metrics, want %d", len(gauges), len(want))
	}
}

func TestCollectRunTimeGauges_RandomValueChanges(t *testing.T) {
	first := CollectRunTimeGauges()["RandomValue"]
	second := CollectRunTimeGauges()["RandomValue"]

	if first == second {
		t.Error("RandomValue did not change between two calls (extremely unlikely if rand is used correctly)")
	}
}
