package runner

import (
	"testing"

	. "github.com/alranel/cino/lib"
)

func TestAssignDevices(t *testing.T) {
	{
		Config.Devices = []Device{
			{FQBN: "one", Features: []string{"bar"}},
			{FQBN: "two", Features: []string{"foo"}},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireFeatures: []string{"foo"}},
				{RequireFeatures: []string{"bar"}},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "two" || devices[1].FQBN != "one" {
			t.Error("Wrong devices returned")
		}
	}

	{
		Config.Devices = []Device{
			{FQBN: "one", Features: []string{"bar"}},
			{FQBN: "two", Features: []string{"foo", "bar"}},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireFeatures: []string{"foo"}},
				{RequireFeatures: []string{"bar"}},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "two" || devices[1].FQBN != "one" {
			t.Error("Wrong devices returned")
		}
	}

	{
		Config.Devices = []Device{
			{FQBN: "one", Features: []string{"foo", "bar"}},
			{FQBN: "two", Features: []string{"bar"}},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireFeatures: []string{"foo"}},
				{RequireFeatures: []string{"bar"}},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "one" || devices[1].FQBN != "two" {
			t.Error("Wrong devices returned")
		}
	}

	{
		Config.Devices = []Device{
			{FQBN: "zero", Features: []string{"foo", "bar", "baz"}},
			{FQBN: "one", Features: []string{"foo"}},
			{FQBN: "two", Features: []string{"foo"}},
			{FQBN: "three", Features: []string{"foo", "bar", "baz"}},
			{FQBN: "four", Features: []string{"foo", "bar"}},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireFeatures: []string{"foo"}},
				{RequireFeatures: []string{"bar"}},
				{RequireFeatures: []string{"baz"}},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "one" || devices[1].FQBN != "three" || devices[2].FQBN != "zero" {
			t.Logf("%v", devices)
			t.Error("Wrong devices returned")
		}
	}

	{
		Config.Devices = []Device{
			{FQBN: "arduino:megaavr:nona4809"},
			{FQBN: "arduino:samd:nano_33_iot"},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireArchitecture: "samd"},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "arduino:samd:nano_33_iot" {
			t.Logf("%v", devices)
			t.Error("Wrong device returned")
		}
	}

	{
		Config.Devices = []Device{
			{FQBN: "arduino:megaavr:nona4809"},
			{FQBN: "arduino:samd:nano_33_iot"},
		}
		test := TestRequirements{
			Sketches: []SketchRequirements{
				{RequireArchitecture: "megaavr"},
			},
		}
		devices := AssignDevices(test)
		if len(devices) != len(test.Sketches) {
			t.Error("Wrong number of devices returned")
		}
		if devices[0].FQBN != "arduino:megaavr:nona4809" {
			t.Logf("%v", devices)
			t.Error("Wrong device returned")
		}
	}
}
