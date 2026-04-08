#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.primitive import Primitive, is_primitive


class TestIsPrimitive:
    def test_string(self) -> None:
        assert is_primitive("hello") is True

    def test_int(self) -> None:
        assert is_primitive(42) is True

    def test_float(self) -> None:
        assert is_primitive(3.14) is True

    def test_bool(self) -> None:
        assert is_primitive(True) is True

    def test_none(self) -> None:
        assert is_primitive(None) is True

    def test_list_is_not_primitive(self) -> None:
        assert is_primitive([1, 2]) is False

    def test_dict_is_not_primitive(self) -> None:
        assert is_primitive({"a": 1}) is False


class TestPrimitiveType:
    def test_accepts_str(self) -> None:
        v: Primitive = "hello"
        assert v == "hello"

    def test_accepts_none(self) -> None:
        v: Primitive = None
        assert v is None
