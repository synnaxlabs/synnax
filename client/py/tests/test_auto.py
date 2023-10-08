import pytest
import synnax as sy
from synnax.simulation.auto import (
    Simulator,
    TankPressure,
    VoltageFull,
    VoltageZero,
    Nominal,
    PreValveStuckOpen,
)


@pytest.mark.auto
class TestSimulator:
    def test_init(self):
        state = {"0": 0, "1": 0, "2": 1}
        nominal = TankPressure("0", "1", "2")
        anomaly = VoltageZero(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)

        assert simmy.state["0"] == 0
        assert simmy.state["1"] == 0
        assert simmy.state["2"] == 1
        return None

    def test_nominal(self):
        state = {"0": 0, "1": 0, "2": 1}
        nominal = TankPressure("0", "1", "2")
        anomaly = Nominal(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 400

        return None

    def test_voltage_zero(self):
        state = {"0": 0, "1": 0, "2": 1}
        nominal = TankPressure("0", "1", "2")
        anomaly = VoltageZero(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == -200

        return None

    def test_voltage_full(self):
        state = {"0": 0, "1": 0, "2": 1}
        nominal = TankPressure("0", "1", "2")
        anomaly = VoltageFull(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 1000

        return None

    def test_prevalve_stuck_open(self):
        state = {"0": 0, "1": 0, "2": 1}
        nominal = TankPressure("0", "1", "2")
        anomaly = PreValveStuckOpen(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 400

        return None
