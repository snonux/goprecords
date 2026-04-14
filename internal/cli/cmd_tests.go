package cli

import "codeberg.org/snonux/goprecords/internal/goprecords"

func runTests() error {
	return goprecords.RunIntegrationTests("./fixtures")
}
