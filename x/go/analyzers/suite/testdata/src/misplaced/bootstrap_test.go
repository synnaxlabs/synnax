package misplaced_test

func RunSpecs(...any) bool       { return true }
func RegisterFailHandler(...any) {}
func Fail(...any)                {}

func bootstrap() {
	RegisterFailHandler(Fail) // want "RegisterFailHandler must live in the suite file misplaced_suite_test.go"
	RunSpecs()                // want "RunSpecs must live in the suite file misplaced_suite_test.go"
}
