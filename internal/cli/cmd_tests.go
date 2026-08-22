package cli

import "github.com/snonux/goprecords/internal/goprecords"

func runTests() error {
	return goprecords.RunIntegrationTests("./fixtures")
}
