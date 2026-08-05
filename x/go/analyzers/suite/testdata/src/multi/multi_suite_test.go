package multi_test

func RunSpecs(...any) bool { return true }

func bootstrap() {
	RunSpecs()
	RunSpecs() // want "package already calls RunSpecs; a package runs one suite"
}
