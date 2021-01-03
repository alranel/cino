package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

type PackageType int

const (
	Sketch PackageType = iota
	Library
	Core
)

type GlobalTestRequirements struct {
	RequireWiring []string `yaml:"require-wiring"`
}

type SketchRequirements struct {
	RequireFQBN         string   `yaml:"require-fqbn"`
	RequireArchitecture string   `yaml:"require-architecture"`
	RequireFeatures     []string `yaml:"require-features"`
}

type TestRequirements struct {
	GlobalTestRequirements
	Sketches []SketchRequirements
}

// TestYML represents the contents of a cino.yml file.
type TestYML struct {
	GlobalTestRequirements `yaml:",inline"`
	Sketches               []testSketch
}

type testSketch struct {
	Dir       string
	Libraries []string
	SketchRequirements
}

// Test represents a directory containing a cino.yml file.
type Test struct {
	TestYML
	Path        string // absolute path to the test directory
	PackagePath string // absolute path to the package containing the test (if any)
	PackageType PackageType
	Status      string // success, failure, skipped
	Output      string
	DeviceFQBNs []string
}

// NewTest instantiates a new Test object.
func NewTest(path, packagePath string, packageType PackageType) (*Test, error) {
	test := &Test{
		Path:        path,
		PackagePath: packagePath,
		PackageType: packageType,
	}

	// Parse the cino.yml file.
	{
		yamlFile, err := ioutil.ReadFile(filepath.Join(test.Path, "cino.yml"))
		if err != nil {
			return nil, fmt.Errorf("error reading cino.yml: %wr", err)
		}
		err = yaml.Unmarshal(yamlFile, &test.TestYML)
		if err != nil {
			return nil, fmt.Errorf("error parsing cino.yml: %wr", err)
		}
	}

	// If cino.yml defines no sketches, create a default one.
	if len(test.Sketches) == 0 {
		test.Sketches = append(test.Sketches, testSketch{Dir: "."})
	} else if len(test.Sketches) == 1 {
		test.Sketches[0].Dir = "."
	} else {
		// Check that all referenced sketches exist
		for _, s := range test.Sketches {
			if s.Dir == "" {
				return nil, fmt.Errorf("Missing sketch directory (dir) for multi-sketch test\n")
			}
			if _, err := os.Stat(filepath.Join(test.Path, s.Dir)); os.IsNotExist(err) {
				return nil, fmt.Errorf("Sketch referenced in cino.yml does not exist: %s\n", s.Dir)
			}
		}
	}

	return test, nil
}

// RelPath returns the test path relative to the repository root.
func (test *Test) RelPath() string {
	path, _ := filepath.Rel(test.PackagePath, test.Path)
	return path
}

func (test *Test) GetRequirements() TestRequirements {
	tr := TestRequirements{GlobalTestRequirements: test.GlobalTestRequirements}
	for _, s := range test.Sketches {
		tr.Sketches = append(tr.Sketches, s.SketchRequirements)
	}
	return tr
}

func (tr *TestRequirements) Clone() (out TestRequirements) {
	out.RequireWiring = append(out.RequireWiring, tr.RequireWiring...)
	for _, s := range tr.Sketches {
		var s2 SketchRequirements
		s2.RequireArchitecture = s.RequireArchitecture
		s2.RequireFQBN = s.RequireFQBN
		s2.RequireFeatures = append(s2.RequireFeatures, s.RequireFeatures...)
		out.Sketches = append(out.Sketches, s2)
	}
	return out
}

func (tr TestRequirements) Equals(tr2 TestRequirements) bool {
	// Normalize requirements' order to make structs comparable
	sort.Strings(tr.RequireWiring)
	for i := range tr.Sketches {
		sort.Strings(tr.Sketches[i].RequireFeatures)
	}
	sort.Strings(tr2.RequireWiring)
	for i := range tr2.Sketches {
		sort.Strings(tr2.Sketches[i].RequireFeatures)
	}
	return cmp.Equal(tr, tr2)
}

/*
func (tr *TestRequirements) Apply(tr2 TestRequirements) {
	tr.RequireWiring = append(tr.RequireWiring, tr2.RequireWiring...)
	for i := range tr.Sketches {
		tr.Sketches[i].RequireArchitecture = tr2.
		s2.RequireArchitecture = s.RequireArchitecture
		s2.RequireFQBN = s.RequireFQBN
		s2.RequireFeatures = append(s2.RequireFeatures, s.RequireFeatures...)
		out.Sketches = append(out.Sketches, s2)
	}
}
*/
