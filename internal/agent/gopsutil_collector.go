package agent

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func CollectGopsutilGauges() (map[string]float64, error) {
	result := make(map[string]float64)

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	result["TotalMemory"] = float64(vm.Total)
	result["FreeMemory"] = float64(vm.Free)

	percentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	for i, p := range percentages {
		result[fmt.Sprintf("CPUutilization%d", i+1)] = p
	}
	return result, nil

}
