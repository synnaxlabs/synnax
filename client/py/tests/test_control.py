import pytest
from synnax.control.auto_sequence import AutoSequence


class TestAutoSequence:
    def test_auto_sequence(self, client):
        with client.auto_sequence() as auto:
            curr = c.copv_pt
            STEP = 100
            while True:
                auto.copv_press_en = True
                auto.wait_until(lambda c: c.copv_pt > curr + STEP)
                curr = c.copv_pt

    def maintain_pressure(
        self,
        auto: AutoSequence,
        pressure: str,
        valve: str,
        target: float,
        tolerance: float = 0.1,
    ):
        while True:
            auto.wait_until(lambda c: c[pressure] > target + tolerance)
            auto[valve] = False
            auto.wait_until(lambda c: c[pressure] < target - tolerance)
            auto[valve] = True
