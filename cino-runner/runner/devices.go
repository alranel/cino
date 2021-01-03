package runner

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/alranel/cino/lib"
	"github.com/thoas/go-funk"
)

func MatchDevice(skreq *SketchRequirements, dev Device) bool {
	if skreq.RequireFQBN != "" && skreq.RequireFQBN != "*" && skreq.RequireFQBN != dev.FQBN {
		return false
	}

	if skreq.RequireArchitecture != "" && skreq.RequireArchitecture != "*" {
		t := strings.SplitN(dev.FQBN, ":", 3)
		if skreq.RequireArchitecture != t[1] {
			return false
		}
	}

	if len(funk.SubtractString(skreq.RequireFeatures, dev.Features)) > 0 {
		return false
	}

	return true
}

// AssignDevices returns an ordered list of devices satisfying the requirements of the test.
func AssignDevices(test TestRequirements) []Device {
	// Check global requirements (our capabilities must be a superset of the job requirements)
	if len(funk.SubtractString(test.RequireWiring, Config.Wiring)) > 0 {
		fmt.Printf("Wiring required by job (%v) don't match our capabilities (%v); skipping\n",
			test.RequireWiring, Config.Wiring)
		return nil
	}

	// If the job requires more devices than we have, this isn't a job for us.
	if len(test.Sketches) > len(Config.Devices) {
		fmt.Printf("Job requires %d, we have %d; skipping\n", len(test.Sketches), len(Config.Devices))
		return nil
	}

	// Check device requirements
	matchingDevices := make(map[int][]int, len(test.Sketches)) // jobDeviceIdx => [ourDeviceIdx, ourDeviceIdx...]
	for i, s := range test.Sketches {
		matchingDevices[i] = []int{}
		// Look for all devices matching the requirements
		for j, device := range Config.Devices {
			if MatchDevice(&s, device) {
				matchingDevices[i] = append(matchingDevices[i], j)
			}
		}
	}
	// Let's try to assign devices in a clever way, prioritizing the ones with fewer matching devices.
	assignedDevices := make([]Device, len(test.Sketches))
	{
		// Sort job devices by number of matching devices.
		sortedKeys := make([]int, len(matchingDevices))
		{
			i := 0
			for k := range matchingDevices {
				sortedKeys[i] = k
				i++
			}
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			a, b := matchingDevices[sortedKeys[i]], matchingDevices[sortedKeys[j]]
			return len(a) < len(b)
		})

		// Go through job devices and assign a device
		for _, jobDeviceIdx := range sortedKeys {
			if len(matchingDevices[jobDeviceIdx]) == 0 {
				fmt.Printf("  no available devices matching job device %d\n", jobDeviceIdx)
				return nil
			}
			assignedDeviceIdx := matchingDevices[jobDeviceIdx][0]
			assignedDevices[jobDeviceIdx] = Config.Devices[assignedDeviceIdx]

			// Make the assigned device unavailble for the next cycles.
			for k, v := range matchingDevices {
				matchingDevices[k] = funk.Subtract(v, []int{assignedDeviceIdx}).([]int)
			}
		}
	}
	return assignedDevices
}
