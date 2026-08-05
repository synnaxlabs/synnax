package orphan_test

func Describe(...any) bool { return true }

var _ = Describe("Orphan", func() {}) // want "never calls RunSpecs; add orphan_suite_test.go"
