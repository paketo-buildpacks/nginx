package nginx_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNGINX(t *testing.T) {
	suite := spec.New("nginx", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("LogEmitter", testLogEmitter)
	suite("Parse", testParser)
	suite("ProfileWriter", testProfileWriter)
	suite.Run(t)
}
