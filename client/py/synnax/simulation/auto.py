import time
from typing import Protocol
import synnax as sy


class SimulatedResponse(Protocol):
    def __init__(self):
        ...

    def step(self, state: dict[str, float]) -> dict[str, float]:
        ...


class Anomaly(Protocol):
    def __init__(self):
        ...

    def step(self, state: dict[str, float]) -> dict[str, float]:
        ...


# All nominal state machines


class TankPressure:
    def __init__(
        self, pt_channel: str, press_valve_channel: str, prevalve_channel: str
    ) -> SimulatedResponse:
        self.step_num = 0
        self.pt = pt_channel
        self.press = press_valve_channel
        self.pre = prevalve_channel

        # state[pt] = 0  # presssure
        # state[press] = 0  # pressvalve, 0 closed, 1 open
        # state[pre] = 1  # prevalve, 0 open, 1 closed

    def step(self, state: dict[str, float]) -> dict[str, float]:
        self.step_num += 1

        if state[self.pt] >= 400:
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


class Nominal:
    def __init__(self, state_machine, error_step: int):
        self.state_machine = state_machine
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int):
        self.state_machine.step(state)
        return state


# Anomaly Implementations


class VoltageZero:
    def __init__(self, state_machine, error_step: int):
        self.state_machine = state_machine
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int):
        if step_num < self.error_step:
            state[self.state_machine.pt] = -200
        self.state_machine.step_num += 1
        return state


class VoltageFull:
    def __init__(self, state_machine, error_step: int):
        self.state_machine = state_machine
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int):
        if step_num < self.error_step:
            state[self.state_machine.pt] = 1000
        self.state_machine.step_num += 1
        return state


class PressValveStuckOpen:
    def __init__(self, state_machine, error_step: int):
        self.state_machine = state_machine
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int):
        if step_num < self.error_step:
            state[self.state_machine.press] == 1
            if state[self.state_machine.pre] == 0:  ##open, pressure stays constant
                state[self.state_machine.pt] += 0
            else:
                state[self.state_machine.pt] += 1
        return self.state_machine.step(state)


class PreValveStuckOpen:
    def __init__(self, state_machine, error_step: int):
        self.state_machine = state_machine
        self.error_step = error_step

    def step(self, state: dict[str, float], step_num: int):
        if step_num < self.error_step:
            if state[self.state_machine.press] == 1:  ##open, pressure stays constant
                state[self.state_machine.pt] += 0
            else:
                state[self.state_machine.pt] -= 1
        return self.state_machine.step(state)


# Base simulator class


class Simulator:
    frequency: sy.Rate.HZ = 100 * sy.Rate.HZ
    state: dict[str, float]
    simulated_responders: list[SimulatedResponse]
    anomaly: Anomaly

    def __init__(
        self,
        input_state,
        input_responders: list[SimulatedResponse],
        input_anomaly: Anomaly,
    ):
        self.state = input_state
        self.state_machines = input_responders
        self.anomaly = input_anomaly

    def run(self):
        start_time = time.time()
        while True:
            time.sleep(1 / self.frequency)
            self.step()

            curr_time = time.time()
            if curr_time - start_time >= 5:
                break

    def step(self):
        for responder in self.state_machines:
            self.state = self.anomaly.step(self.state, responder.step_num)
