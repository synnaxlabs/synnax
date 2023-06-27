import time
from typing import Protocol

import synnax as sy

client = sy.Synnax()


class SimulatedResponse(Protocol):
    def __int__(self):
        ...

    def step(self, state: dict[str, float]) -> dict[str, float]:
        ...


class TankPressure:
    valve_channel: str
    pressure_channel: str

    def _(self,
            valve_channel: str,
            pressure_channel: str) -> TankPressure:
        return self

    def step(self, state: dict[str, float]) -> dict[str, float]:
        if state[self.valve_channel]:
            state["pressure"] += 1
        else:
            state["pressure"] -= 1
        return state


class Simulator:
    frequency: sy.Rate.HZ = 100 * sy.Rate.HZ
    state: dict[str, float]
    simulated_responders: list[SimulatedResponse]

    def __init__(self):
        ...

    def run(self):
        while True:
            time.sleep(1 / self.frequency)
            self.step()

    def step(self):
        for responder in self.simulated_responders:
            self.state = responder.step(self.state)


with client.control(
    "TPC",
    simulator=Simulator(
        respones=[
            TankPressure("vlv1", "pressure"),
            COPVTemp("vlv2", "pressure2"),
        ]
    )
) as auto:
    while True:
        auto.vlv1 = True
        auto.wait_until(lambda c: c.pressure > 100 or c.pressure2 > 105)
        auto.vlv1 = False
        auto.wait_until(lambda c: c.pressure < 50)
