#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import freighter
from x.exceptions import Error, ExceptionPayload, decode_exception


class TestDecode:
    def test_eof(self) -> None:
        """Should decode a freighter EOF payload."""
        pld = ExceptionPayload(type=freighter.EOF.TYPE, data="EOF")
        assert isinstance(decode_exception(pld), freighter.EOF)

    def test_unknown_type(self) -> None:
        """Should decode an unknown freighter type into a generic Synnax error."""
        pld = ExceptionPayload(type="freighter.unknown", data="unknown error")
        exc = decode_exception(pld)
        assert type(exc) is Error
        assert str(exc) == "unknown error"
