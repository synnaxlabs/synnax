#
#
# class SampleError(Exception):
#     data: str
#
#     def __init__(self, data: str) -> None:
#         self.data = data
#
#     @classmethod
#     def type(cls) -> str:
#         return "sample"
#
#     @classmethod
#     def decode(cls, encoded: str) -> Exception:
#         return SampleError(encoded)
#
#     def encode(self) -> str:
#         return self.data
#
#
# class TestErrorRegistry:
#     def test_register_does_not_implement_error_protocol(self):
#         reg = ErrorRegistry()
#         with pytest.raises(TypeError):
#             reg.register(Exception())
#
#     def test_register(self):
#         reg = ErrorRegistry()
#         reg.register(SampleError)
#         assert SampleError.type() in reg.entries
#
#     def test_encode_decode(self):
#         reg = ErrorRegistry()
#         reg.register(SampleError)
#         error = SampleError("error")
#         encoded = reg.encode(error)
#         decoded = reg.decode(encoded)
#         assert decoded.data == error.data
#
#     def test_encode_decode_not_in_registry(self):
#         err = ErrorRegistry()
#         encoded = err.encode(Exception("error"))
#         decoded = err.decode(encoded)
#         assert isinstance(decoded, Exception)
