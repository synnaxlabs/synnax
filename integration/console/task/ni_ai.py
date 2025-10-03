#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Optional

from playwright.sync_api import Page

from .task import Task

if TYPE_CHECKING:
    from console.console import Console


class NiAi(Task):
    """NI Analog Input Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Analog Read Task"
        self.pluto_label = ".ni_ai_somethingsomething"

    def new(self) -> str:
        """Create a new NI AI task page and set console.task to this instance."""
        result = super().new()
        self.console.task = self
        return result

    def set_parameters(
        self,
        task_name: Optional[str] = None,
        sample_rate: Optional[float] = None,
        stream_rate: Optional[float] = None,
        data_saving: Optional[bool] = None,
        auto_start: Optional[bool] = None,
    ) -> None:
        """
        Set the parameters for the task.

        Args:
            sample_rate: The sample rate for the task.
            stream_rate: The stream rate for the task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
        """
        console = self.console

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start
        )

        if sample_rate is not None:
            console.fill_input_field("Sample Rate", str(sample_rate))

        if stream_rate is not None:
            console.fill_input_field("Stream Rate", str(stream_rate))

