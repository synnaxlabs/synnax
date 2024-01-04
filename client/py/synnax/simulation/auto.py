#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from typing import Protocol
import synnax as sy


class SimulatedResponse(Protocol):
    def __init__(self):
        ...

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        ...


# All nominal state machines


class TankPressure:
    def __init__(
        self, pt_channel: str, press_valve_channel: str, prevalve_channel: str
    ) -> SimulatedResponse:
        self.pt = pt_channel
        self.press = press_valve_channel
        self.pre = prevalve_channel

        # state[pt] = 0  # presssure
        # state[press] = 0  # pressvalve, 0 closed, 1 open
        # state[pre] = 1  # prevalve, 0 open, 1 closed

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        if state[self.pt] >= 500:
            state[self.press] = 0
        else:
            state[self.press] = 1

        if state[self.press] == 1 and state[self.pre] == 1:
            state[self.pt] += 1
        elif state[self.press] == 0 and state[self.pre] == 0:
            state[self.pt] -= 1
        else:
            state[self.pt] += 0

        return state


class TankTemperature:
    def __init__(
        self, tc_channel: str, press_valve_channel: str, prevalve_channel: str
    ) -> SimulatedResponse:
        self.tc = tc_channel
        self.press = press_valve_channel
        self.pre = prevalve_channel

        # state[tc] = 0  # temperature
        # state[press] = 0  # pressvalve, 0 closed, 1 open
        # state[pre] = 1  # prevalve, 0 open, 1 closed

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        if state[self.press] == 1 and state[self.pre] == 1:
            state[self.tc] += 1
        elif state[self.press] == 0 and state[self.pre] == 0:
            state[self.tc] -= 1
        else:
            state[self.tc] += 0

        return state


# Anomaly Implementations


class VoltageFailure:
    def __init__(
        self,
        state_machine: SimulatedResponse,
        pt_channel: str,
        error_type: bool,
        error_step: int,
    ) -> SimulatedResponse:
        self.state_machine = state_machine
        self.pt = pt_channel
        self.error_type = error_type
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        if step_num > self.error_step:
            if self.error_type:
                state[self.pt] = 1000
            else:
                state[self.pt] = -200
        else:
            self.state_machine.step(state, step_num)
        return state


class ValveFailure:
    def __init__(
        self,
        state_machine: SimulatedResponse,
        valve_channel: str,
        error_step: int,
    ) -> SimulatedResponse:
        self.state_machine = state_machine
        self.valve_channel = valve_channel
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        if step_num > self.error_step:
            state[self.valve_channel] = 0
        else:
            self.state_machine.step(state, step_num)
        return state


class TCFailure:
    def __init__(
        self,
        state_machine: SimulatedResponse,
        tc_channel: str,
        error_step: int,
    ) -> SimulatedResponse:
        self.state_machine = state_machine
        self.tc_channel = tc_channel
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int) -> dict[str, float]:
        if step_num > self.error_step:
            state[self.tc_channel] = 0
        else:
            self.state_machine.step(state, step_num)
        return state


# Base simulator class


class Simulator:
    frequency: sy.Rate.HZ = 100 * sy.Rate.HZ
    state: dict[str, float]
    simulated_responders: list[SimulatedResponse]

    def __init__(
        self,
        input_state,
        simulated_responders,
    ):
        self.state = input_state
        self.stepNum = 0
        self.simulated_responders = simulated_responders

    def run(self):
        start_time = time.time()
        while True:
            time.sleep(1 / self.frequency)
            self.step()

            curr_time = time.time()
            if self.stepNum == 500:
                break

            self.stepNum += 1

    def step(self):
        for simulated_responders in self.simulated_responders:
            self.state = simulated_responders.step(self.state, self.stepNum)
