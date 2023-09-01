
from typing import Callable

class Tank:
    pt_chan: str
    press_chan: str
    vent_chan: str
    press_calc_method: Callable[[float], float]

    def __init__(self, pt_channel: str):
        ...

    def step(self, state: dict[str, float]) -> dict[str, float]:
        press_valve_value = state[self.press_chan]
        if (press_valve_value > 0.5):
            state[self.pt_chan] = self.pt_chan + self.press_calc_method(press_valve_value)
        else:
            state[self.pt_chan] = self.pt_chan + self.press_calc_method(press_valve_value)

