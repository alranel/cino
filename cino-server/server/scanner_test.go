package server

import (
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"testing"

	. "github.com/alranel/cino/lib"
	"github.com/google/go-cmp/cmp"
)

func TestPerm(t *testing.T) {
	set := []string{"foo", "bar", "baz"}
	{
		perm := Perm(set, 1)
		if len(perm) != int(math.Pow(float64(len(set)), 1)) {
			t.Error("Wrong number of items")
		}
		if !cmp.Equal(perm[0], []string{"foo"}) || !cmp.Equal(perm[len(perm)-1], []string{"baz"}) {
			fmt.Printf("%v\n", perm)
			t.Error("Bad permutation when k == 1")
		}
	}
	{
		perm := Perm(set, 2)
		if len(perm) != int(math.Pow(float64(len(set)), 2)) {
			t.Error("Wrong number of items")
		}
		if !cmp.Equal(perm[0], []string{"foo", "foo"}) || !cmp.Equal(perm[len(perm)-1], []string{"baz", "baz"}) {
			fmt.Printf("%v\n", perm)
			t.Error("Bad permutation when k < len(set)")
		}
	}
	{
		perm := Perm(set, 3)
		if len(perm) != int(math.Pow(float64(len(set)), 3)) {
			t.Error("Wrong number of items")
		}
		if !cmp.Equal(perm[0], []string{"foo", "foo", "foo"}) || !cmp.Equal(perm[len(perm)-1], []string{"baz", "baz", "baz"}) {
			fmt.Printf("%v\n", perm)
			t.Error("Bad permutation when k == len(set)")
		}
	}
	{
		perm := Perm(set, 4)
		if len(perm) != int(math.Pow(float64(len(set)), 4)) {
			t.Error("Wrong number of items")
		}
		if !cmp.Equal(perm[0], []string{"foo", "foo", "foo", "foo"}) || !cmp.Equal(perm[len(perm)-1], []string{"baz", "baz", "baz", "baz"}) {
			fmt.Printf("%v\n", perm)
			t.Error("Bad permutation when k > len(set)")
		}
	}
}

func TestRepeatByFQBNs(t *testing.T) {
	// Note: this function does not currently check the "Original" key of the returned matrices
	checkResult := func(tmpl []TestRequirements, fqbns []string, expected []TestRequirements) {
		uniqueResult := uniqRequirements(RepeatByFQBNs(tmpl, fqbns))
		var result2 []TestRequirements
		for _, trm := range uniqueResult {
			result2 = append(result2, trm.Effective)
		}
		if !cmp.Equal(result2, expected) {
			t.Error("Did not get expected test plan")
			t.Logf("Got (%d): %+v\n", len(result2), result2)
			t.Logf("Expected (%d): %+v\n", len(expected), expected)
			t.Log(string(debug.Stack()))
		}
	}

	{
		tmpl := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "", RequireFeatures: []string{"foo", "bar"}},
				},
			},
		}
		fqbns := []string{"arduino:avr:uno", "arduino:avr:nano"}
		expected := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"bar", "foo"}},
				},
			},
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"bar", "foo"}},
				},
			},
		}
		checkResult(tmpl, fqbns, expected)
	}

	{
		tmpl := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "", RequireFeatures: []string{"foo", "bar"}},
					{RequireFQBN: "", RequireFeatures: []string{"baz"}},
				},
			},
		}
		fqbns := []string{"arduino:avr:uno", "arduino:avr:nano"}
		expected := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"baz"}},
				},
			},
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"baz"}},
				},
			},
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"baz"}},
				},
			},
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"baz"}},
				},
			},
		}
		checkResult(tmpl, fqbns, expected)
	}

	{
		tmpl := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "", RequireFeatures: []string{"foo", "bar"}},
					{RequireFQBN: "*", RequireFeatures: []string{"baz"}},
				},
			},
		}
		fqbns := []string{"arduino:avr:uno", "arduino:avr:nano"}
		expected := []TestRequirements{
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:uno", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "*", RequireFeatures: []string{"baz"}},
				},
			},
			{
				GlobalTestRequirements: GlobalTestRequirements{RequireWiring: []string{"i2c"}},
				Sketches: []SketchRequirements{
					{RequireFQBN: "arduino:avr:nano", RequireFeatures: []string{"bar", "foo"}},
					{RequireFQBN: "*", RequireFeatures: []string{"baz"}},
				},
			},
		}
		checkResult(tmpl, fqbns, expected)
	}
}

func TestGetArchitecturesFromLibrary(t *testing.T) {
	repoDir, err := CloneRepo("https://github.com/arduino-libraries/Servo.git", "HEAD")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(repoDir)
	result, err := getArchitecturesFromLibrary(repoDir)
	if err != nil {
		panic(err)
	}
	expected := []string{"avr", "megaavr", "sam", "samd", "nrf52", "stm32f4", "mbed"}
	if !cmp.Equal(result, expected) {
		t.Error("Did not get expected values")
		t.Logf("(got %v, expected %v)", result, expected)
	}
}

func TestGetBoardsFromCore(t *testing.T) {
	repoDir, err := CloneRepo("https://github.com/arduino/ArduinoCore-avr.git", "HEAD")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(repoDir)
	result, err := getBoardsFromCore(repoDir)
	if err != nil {
		panic(err)
	}
	expected := []string{"yun", "uno", "diecimila", "nano", "mega", "megaADK", "leonardo", "leonardoeth", "micro", "esplora", "mini", "ethernet", "fio", "bt", "LilyPadUSB", "lilypad", "pro", "atmegang", "robotControl", "robotMotor", "gemma", "circuitplay32u4cat", "yunmini", "chiwawa", "one", "unowifi"}
	if !cmp.Equal(result, expected) {
		t.Error("Did not get expected values")
		t.Logf("(got %v, expected %v)", result, expected)
	}
}
