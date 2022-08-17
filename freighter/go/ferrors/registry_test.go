package ferrors_test

//type SimpleError struct {
//	data string
//}
//
//func (s SimpleError) Provider() string { return s.data }
//
//func (s SimpleError) Type() errors.Type { return "simple" }
//
//func (s SimpleError) Encode() string { return s.data }
//
//func (s SimpleError) Decode(data string) error { return SimpleError{data} }
//
//var _ = Describe("Errors", func() {
//	Describe("register", func() {
//		It("should register an error type", func() {
//			Expect(func() {
//				r.Register(SimpleError{})
//			}).ToNot(Panic())
//		})
//	})
//	Describe("encode", func() {
//		It("Should encode nil error into a payload with type Nil", func() {
//			r.Register(SimpleError{})
//			p := r.Encode(nil)
//			Expect(p.Type).To(Equal(errors.Nil))
//			Expect(p.Payload).To(Equal(""))
//		})
//		It("Should encode a simple error into a payload with type simple", func() {
//			r.Register(SimpleError{})
//			p := r.Encode(SimpleError{data: "hello"})
//			Expect(p.Type).To(Equal(errors.Type("simple")))
//			Expect(p.Payload).To(Equal("hello"))
//		})
//		It("Should encode an unknown error into a payload with type Unknown", func() {
//			r.Register(SimpleError{})
//			p := r.Encode(roacherrors.New("hello"))
//			Expect(p.Type).To(Equal(errors.Unknown))
//			Expect(p.Payload).To(Equal("hello"))
//		})
//	})
//})
