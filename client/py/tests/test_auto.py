
import pytest
import numpy as np
import pandas as pd

import synnax as sy
from synnax.simulation.auto import Simulator
from synnax.simulation.auto import TankPressure
from synnax.simulation.auto import VoltageZero
from synnax.simulation.auto import VoltageFull
from synnax.simulation.auto import Nominal
from synnax.simulation.auto import PreValveStuckOpen

@pytest.mark.auto
class TestSimulator:   
    def test_init(self):
        state = {}
        nominal = TankPressure(state)
        anomaly = VoltageZero(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        
        assert simmy.state["0"] == 0
        assert simmy.state["1"] == 0
        assert simmy.state["2"] == 1

        return None
    
    def test_nominal(self):
        state = {}
        nominal = TankPressure(state)
        anomaly = Nominal(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 400

        return None
    
    def test_voltage_zero(self):
        state = {}
        nominal = TankPressure(state)
        anomaly = VoltageZero(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == -200

        return None
    
    def test_voltage_full(self):
        state = {}
        nominal = TankPressure(state)
        anomaly = VoltageFull(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 1000

        return None

    def test_prevalve_stuck_open(self):
        state = {}
        nominal = TankPressure(state)
        anomaly = PreValveStuckOpen(nominal, 100)
        simmy = Simulator(state, nominal, anomaly)
        simmy.run()

        assert simmy.state["0"] == 400

        return None
