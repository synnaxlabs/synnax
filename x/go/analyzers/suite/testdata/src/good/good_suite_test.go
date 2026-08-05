package good_test

// Local stubs stand in for the dot-imported Ginkgo symbols; the analyzer matches by
// name, so real imports are unnecessary.
func RunSpecs(...any) bool        { return true }
func RegisterFailHandler(...any)  {}
func Describe(...any) bool        { return true }
func Fail(...any)                 {}

func bootstrap() {
	RegisterFailHandler(Fail)
	RunSpecs()
}
