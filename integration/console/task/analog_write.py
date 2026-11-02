#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import Concatenate, ParamSpec, TypeVar

from console.layout import LayoutClient
from console.page import PageType
from console.task.channels.analog_output import Current, Voltage
from console.task.ni import NITask

_Channel = TypeVar("_Channel", bound=Voltage | Current)
_P = ParamSpec("_P")


class AnalogWrite(NITask):
    """NI Analog Write/Output Task automation interface."""

    page_type: PageType = "NI analog write task"
    pluto_label: str = ".console-task-configure--ni_analog_write"

    def add_channel(
        self,
        name: str,
        chan_type: Callable[Concatenate[LayoutClient, str, str, _P], _Channel],
        device: str,
        dev_name: str | None = None,
        *args: _P.args,
        **kwargs: _P.kwargs,
    ) -> _Channel:
        """
        Add an analog write channel to the task.

        Args:
            name: Channel name
            chan_type: Channel class, which must be valid for analog write tasks
            device: Device identifier
            dev_name: Optional device name
            *args, **kwargs: Channel-specific configuration

        Returns:
            The created channel instance
        """
        return self._add_channel(name, chan_type, device, dev_name, *args, **kwargs)

    def set_parameters(
        self,
        *,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
        sample_rate: float | None = None,
        stream_rate: float | None = None,
        state_update_rate: float | None = None,
    ) -> None:
        """
        Set the parameters for the NI AO task.

        Args:
            task_name: The name of the task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
            sample_rate: The sample rate for the AO task.
            stream_rate: The stream rate for the AO task.
            state_update_rate: The state update rate for the AO task.
        """

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
        )

        if state_update_rate is not None:
            self.layout.fill_input_field("State update rate", str(state_update_rate))
