#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from sedosaur_request import Request, parse

NONE = Request(requested=False, repeat=False)
PLAIN = Request(requested=True, repeat=False)
REPEAT = Request(requested=True, repeat=True)


class TestParse:
    @pytest.mark.parametrize(
        ("body", "expected"),
        [
            ("@sedosaur review", PLAIN),
            ("  @Sedosaur Review  ", PLAIN),
            ("@sedosaur review repeat=true", REPEAT),
            ("Pushed a fix.\n@sedosaur review repeat=true", REPEAT),
            ("@sedosaur review repeat=false", PLAIN),
            ("@sedosaur review repeat=trueish", PLAIN),
            ("@sedosaur reviewer repeat=true", NONE),
            ("Thanks @sedosaur review repeat=true", NONE),
            ("@greptile review", NONE),
            ("", NONE),
        ],
    )
    def test_should_read_a_request_line(self, body: str, expected: Request) -> None:
        assert parse(body) == expected

    def test_should_ignore_a_quote(self) -> None:
        body = "> comment `@sedosaur review repeat=true`.\nThanks."
        assert parse(body) == NONE

    def test_should_ignore_a_quoted_request(self) -> None:
        assert parse("> @sedosaur review repeat=true") == NONE

    def test_should_ignore_a_code_span(self) -> None:
        assert parse("`@sedosaur review repeat=true`") == NONE

    def test_should_ignore_repeat_in_a_code_span(self) -> None:
        assert parse("@sedosaur review `repeat=true`") == PLAIN

    def test_should_ignore_a_code_fence(self) -> None:
        body = "```\n@sedosaur review repeat=true\n```\n~~~\n@sedosaur review\n~~~"
        assert parse(body) == NONE

    def test_should_read_a_request_after_a_code_fence(self) -> None:
        body = "```\n@sedosaur review\n```\n@sedosaur review repeat=true"
        assert parse(body) == REPEAT
