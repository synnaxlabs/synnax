#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

from console.task_page import TaskPage


class FakeStatusElement:
    def __init__(self, classes: str | None, text: str) -> None:
        self.classes = classes
        self.text = text

    def get_attribute(self, name: str) -> str | None:
        return self.classes

    def inner_text(self) -> str:
        return self.text


class FakeTaskPage(TaskPage):
    def __init__(self, classes: str | None, text: str) -> None:
        self.element = FakeStatusElement(classes, text)

    def _status_element(self) -> FakeStatusElement:
        return self.element


class TestStatus:
    def test_should_read_the_variant_and_message(self) -> None:
        status = FakeTaskPage("lyra-text lyra--status-success", "Task running").status()
        assert status.variant == "success"
        assert status.message == "Task running"

    def test_should_reject_a_box_without_a_variant_class(self) -> None:
        with pytest.raises(ValidationError):
            FakeTaskPage("lyra-text", "Task running").status()

    def test_should_reject_a_box_without_a_class(self) -> None:
        with pytest.raises(ValidationError):
            FakeTaskPage(None, "Task running").status()

    def test_should_reject_an_unknown_variant(self) -> None:
        with pytest.raises(ValidationError):
            FakeTaskPage("lyra--status-unknown", "Task running").status()
