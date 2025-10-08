#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Optional

from playwright.sync_api import Page

from .task import Task

if TYPE_CHECKING:
    from console.console import Console


class NIAnalogRead(Task):
    """NI Analog Read/Input Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Analog Read Task"

    def new(self) -> str:
        """Create a new NI AI task page and set console.task to this instance."""
        result = super().new()
        self.console.task = self
        return result

    def set_parameters(
        self,
        task_name: Optional[str] = None,
        data_saving: Optional[bool] = None,
        auto_start: Optional[bool] = None,
        **kwargs: Any,
    ) -> None:
        """
        Set the parameters for the NI AI task.

        Args:
            task_name: The name of the task.
            sample_rate: The sample rate for the AI task.
            stream_rate: The stream rate for the AI task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
            **kwargs: Additional parameters.
        """
        sample_rate = kwargs.pop("sample_rate", None)
        stream_rate = kwargs.pop("stream_rate", None)

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start,
            **kwargs,
        )

        if sample_rate is not None:
            self.console.fill_input_field("Sample Rate", str(sample_rate))

        if stream_rate is not None:
            self.console.fill_input_field("Stream Rate", str(stream_rate))
