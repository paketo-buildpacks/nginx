package nginx_test

import (
	"testing"

	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNGINX(t *testing.T) {
	format.MaxLength = 0
	suite := spec.New("nginx", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Configuration", testConfiguration)
	suite("DefaultConfigGenerator", testDefaultConfigGenerator)
	suite("Detect", testDetect)
	suite("Parse", testParser)
	suite.Run(t)
}
