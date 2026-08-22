package cli

import (
	"fmt"

	"github.com/snonux/goprecords/internal/version"
)

func Execute(args []string) error {
	if len(args) > 0 && (args[0] == "-version" || args[0] == "--version") {
		fmt.Println(version.Tag)
		return nil
	}
	if len(args) >= 1 && (args[0] == "--create-client-key" || args[0] == "-create-client-key") {
		if len(args) < 2 {
			return fmt.Errorf("create-client-key: hostname required")
		}
		return runCreateClientKey(args[1], args[2:])
	}
	if len(args) > 0 && (args[0] == "-daemon" || args[0] == "--daemon") {
		return runDaemon(args[1:])
	}

	if len(args) == 0 {
		return runReportFromFiles(nil)
	}

	switch args[0] {
	case "import":
		return runImport(args[1:])
	case "query":
		return runQuery(args[1:])
	case "exclude":
		return runExclude(args[1:])
	case "unexclude":
		return runUnexclude(args[1:])
	case "list-excluded":
		return runListExcluded(args[1:])
	case "classify":
		return runClassify(args[1:])
	case "list-classes":
		return runListClasses(args[1:])
	case "test":
		return runTests()
	default:
		return runReportFromFiles(args)
	}
}
