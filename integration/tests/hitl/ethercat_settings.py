#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT read tasks that set up the inputs of a 6xSTG in Synnax-only mode on the HITL
station. The Driver reads the settings of a slave again when its bus stops, so each test
checks the values that the card holds after its tasks stop."""

from pydantic import BaseModel

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import create_channel, create_index, delete_channels
from tests.hitl.ethercat import START_TIMEOUT, STGS, prepare_bus

CHANGED = ("Range", "Low-pass filter")
"""The settings of an input that each test changes."""

_READ_BACK_TIMEOUT = 30 * sy.TimeSpan.SECOND
_DATA_TIMEOUT = 10 * sy.TimeSpan.SECOND


class _Option(BaseModel):
    value: float


class _Condition(BaseModel):
    index: int
    sub_index: int
    values: list[float] | None = None
    above: float | None = None


class _Field(BaseModel):
    index: int
    sub_index: int
    name: str
    data_type: str
    value: float | None = None
    options: list[_Option] = []
    visible_when: list[_Condition] = []


class _Entry(BaseModel):
    index: int
    sub_index: int
    only_read_by: str | None = None


class _Input(BaseModel):
    entries: list[_Entry]
    settings: list[_Field]


class _Mode(BaseModel):
    shared: bool


class _Settings(BaseModel):
    channels: list[_Input]
    mode: _Mode | None = None


class _Properties(BaseModel):
    pdos: dict[str, list[sy.ethercat.PDOEntry]] = {}
    settings: _Settings | None = None


def _held(fields: list[_Field], index: int, sub_index: int) -> float | None:
    """Returns the value that the card held at the address at the last scan."""
    return next(
        (f.value for f in fields if f.index == index and f.sub_index == sub_index),
        None,
    )


def _visible(field: _Field, fields: list[_Field]) -> bool:
    for c in field.visible_when:
        held = _held(fields, c.index, c.sub_index)
        if held is None:
            return False
        if c.values is not None and held not in c.values:
            return False
        if c.values is None and (c.above is None or held <= c.above):
            return False
    return True


def _parameter(field: _Field, value: float) -> sy.ethercat.StartupParameter:
    return sy.ethercat.StartupParameter(
        index=field.index,
        sub_index=field.sub_index,
        data_type=sy.DataType(field.data_type),
        value=value,
    )


def _find_standalone(slaves: dict[int, sy.Device]) -> sy.Device:
    """Returns the first 6xSTG that only Synnax sets up.

    :raises AssertionError: If each 6xSTG is set up for DewesoftX.
    """
    for pos in STGS:
        settings = _Properties.model_validate(slaves[pos].properties).settings
        if settings is not None and (settings.mode is None or not settings.mode.shared):
            return slaves[pos]
    raise AssertionError(
        f"Expected a 6xSTG in Synnax-only mode at position {STGS[0]} or {STGS[1]}. "
        "Switch one in its device settings, then restart its power."
    )


class EtherCATSettings(TestCase):
    """Changes the range and the low-pass filter of input 1 of a 6xSTG from a task that
    starts the bus, and of input 2 from a task that joins the running bus. Then puts
    back the values that the card held."""

    stg: sy.Device
    originals: dict[int, list[sy.ethercat.StartupParameter]]
    """The values that each changed input held before the tests, by input."""
    tasks: list[sy.ethercat.ReadTask]

    def setup(self) -> None:
        self.originals = {}
        self.tasks = []
        self.stg = _find_standalone(prepare_bus(self.client))
        self.log(f"Using the 6xSTG at position {self.stg.properties['position']}")

    def run(self) -> None:
        self.test_settings_at_start()
        self.test_settings_on_running_bus()

    def teardown(self) -> None:
        with self._try_to("put back the settings of the 6xSTG"):
            self._restore()
        for tsk in self.tasks:
            with self._try_to(f"delete task {tsk.name}"):
                self.client.tasks.delete(tsk.key)
        with self._try_to("delete the task channels"):
            delete_channels(
                self.client,
                [ch.channel for tsk in self.tasks for ch in tsk.config.channels],
            )
        super().teardown()

    def test_settings_at_start(self) -> None:
        """Checks that a task that starts the bus sets up the input."""
        self.log("Testing: Settings at the start of the bus")
        parameters = self._change(0)
        tsk = self._create_task("At Start", 0, parameters)
        with tsk.run(timeout=START_TIMEOUT):
            self._await_data(tsk)
        self._await_values(0, parameters)

    def test_settings_on_running_bus(self) -> None:
        """Checks that a task that joins a running bus sets up the input, which proves
        that a 6xSTG takes the values in OP."""
        self.log("Testing: Settings on a running bus")
        first = self._create_task("Running", 0, [])
        parameters = self._change(1)
        second = self._create_task("Joining", 1, parameters)
        with first.run(timeout=START_TIMEOUT):
            self._await_data(first)
            with second.run(timeout=START_TIMEOUT):
                self._await_data(second)
        self._await_values(1, parameters)

    def _input(self, n: int) -> _Input:
        """Returns input n of the 6xSTG, from 0, with the values of the last scan."""
        dev = self.client.devices.retrieve(key=self.stg.key)
        settings = _Properties.model_validate(dev.properties).settings
        if settings is None:
            raise AssertionError(f"Expected the settings of {dev.name}")
        return settings.channels[n]

    def _change(self, n: int) -> list[sy.ethercat.StartupParameter]:
        """Returns a value for each setting in CHANGED of input n that differs from the
        value of the card, and adds the value of the card to the originals."""
        fields = self._input(n).settings
        parameters: list[sy.ethercat.StartupParameter] = []
        for name in CHANGED:
            field = next(
                (f for f in fields if f.name == name and _visible(f, fields)), None
            )
            if field is None or field.value is None:
                raise AssertionError(f"Expected input {n + 1} to show {name}")
            other = next(
                (o.value for o in field.options if o.value != field.value), None
            )
            if other is None:
                raise AssertionError(
                    f"Expected {name} of input {n + 1} to have options"
                )
            self.originals.setdefault(n, []).append(_parameter(field, field.value))
            parameters.append(_parameter(field, other))
        return parameters

    def _create_task(
        self, label: str, n: int, parameters: list[sy.ethercat.StartupParameter]
    ) -> sy.ethercat.ReadTask:
        """Returns a configured task that reads input n with the parameters."""
        entry = next(e for e in self._input(n).entries if e.only_read_by is None)
        pdo = next(
            p
            for p in _Properties.model_validate(self.stg.properties).pdos["inputs"]
            if p.index == entry.index and p.sub_index == entry.sub_index
        )
        data_type = sy.DataType(pdo.data_type)
        # Two tasks that run at once cannot write one index.
        name = f"hitl_ethercat_settings_input_{n + 1}"
        index = create_index(self.client, f"{name}_time")
        tsk = sy.ethercat.ReadTask(
            name=f"HITL EtherCAT Settings {label}",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            channels=[
                sy.ethercat.EntryReadChannel(
                    device=self.stg.key,
                    index=entry.index,
                    sub_index=entry.sub_index,
                    data_type=data_type,
                    channel=create_channel(
                        self.client,
                        name=name,
                        data_type=data_type,
                        index=index.key,
                    ),
                )
            ],
            startup_parameters=(
                [
                    sy.ethercat.SlaveStartupParameters(
                        device=self.stg.key, parameters=parameters
                    )
                ]
                if parameters
                else []
            ),
        )
        self.client.tasks.configure(tsk)
        self.tasks.append(tsk)
        return tsk

    def _await_data(self, tsk: sy.ethercat.ReadTask) -> None:
        key = tsk.config.channels[0].channel
        with self.client.open_streamer([key]) as streamer:
            frame = streamer.read(timeout=_DATA_TIMEOUT)
        if frame is None or len(frame[key]) == 0:
            raise AssertionError(f"Expected task {tsk.name} to read data")

    def _await_values(
        self, n: int, parameters: list[sy.ethercat.StartupParameter]
    ) -> None:
        """Waits until the scan reads the values of the parameters from input n."""
        held: list[float | None] = []

        def done() -> bool | None:
            nonlocal held
            fields = self._input(n).settings
            held = [_held(fields, p.index, p.sub_index) for p in parameters]
            return True if held == [p.value for p in parameters] else None

        if sy.poll(done, timeout=_READ_BACK_TIMEOUT, interval=0.5) is None:
            raise AssertionError(
                f"Expected input {n + 1} of the 6xSTG to hold "
                f"{[p.value for p in parameters]}, but it holds {held}"
            )

    def _restore(self) -> None:
        """Puts back the originals from a task that starts the bus."""
        if not self.originals:
            return
        parameters = [p for ps in self.originals.values() for p in ps]
        with self._create_task("Restore", 0, parameters).run(timeout=START_TIMEOUT):
            pass
        for n, originals in self.originals.items():
            self._await_values(n, originals)
