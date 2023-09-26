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
    def step(self, state: dict[str, float], errorStep: int) -> dict[str, float]:
        ...

# All nominal state machines

class TankPressure:


    def __init__(self, state) -> SimulatedResponse:
        self.stepNum = 0
        state["0"] = 0 #presssure
        state["1"] = 0 #pressvalve, 0 closed, 1 open
        state["2"] = 1 #prevalve, 0 open, 1 closed
    
    #pressValve normally closed
    #preValve normally open
    #normal == 0

    def step(self, state: dict[str, float]) -> dict[str, float]:

        self.stepNum += 1

        if state["0"] >= 400:
            state["1"] = 0
        else:
            state["1"] = 1

        if state["1"] == 1 and state["2"] == 1:
            state["0"] += 1
        elif state["1"] == 0 and state["2"] == 0:
            state["0"] -= 1
        else:
            state["0"] += 0

        return state

class Nominal:

    def __init__(self, stateMachine, errorStep: int):
        self.stateMachine = stateMachine
        self.errorStep = errorStep

    def step(self, state: dict[str,float], stepNum: int):
        self.stateMachine.step(state)
        return state
    
# Anomaly Implementations

class VoltageZero:

    def __init__(self, stateMachine, errorStep: int):
        self.stateMachine = stateMachine
        self.errorStep = errorStep

    def step(self, state: dict[str,float], stepNum: int):
        if(stepNum < self.errorStep):
            state["0"] = -200
        self.stateMachine.stepNum += 1
        return state
    
class VoltageFull:

    def __init__(self, stateMachine, errorStep: int):
        self.stateMachine = stateMachine
        self.errorStep = errorStep

    def step(self, state: dict[str,float], stepNum: int):
        if(stepNum < self.errorStep):
            state["0"] = 1000
        self.stateMachine.stepNum += 1
        return state
     
class PressValveStuckOpen:
    
    def __init__(self, stateMachine, errorStep: int):
        self.stateMachine = stateMachine
        self.errorStep = errorStep

    def step(self, state: dict[str,float], stepNum: int):
        if(stepNum < self.errorStep):
            state["1"] == 1
            if state["2"] == 0: ##open, pressure stays constant
                state["0"] += 0
            else:
                state["0"] += 1
        return self.stateMachine.step(state)
    
class PreValveStuckOpen:
    
    def __init__(self, stateMachine, errorStep: int):
        self.stateMachine = stateMachine
        self.errorStep = errorStep

    def step(self, state: dict[str,float], stepNum: int):
        if(stepNum < self.errorStep):
            if state["1"] == 1: ##open, pressure stays constant
                state["0"] += 0
            else:
                state["0"] -= 1
        return self.stateMachine.step(state)

# Base simulator class

class Simulator:
    frequency: sy.Rate.HZ = 100 * sy.Rate.HZ
    state: dict[str, float]
    state_machine: SimulatedResponse()
    anomaly: Anomaly()

    def __init__(self, inputState, inputResponder: SimulatedResponse(), inputAnomaly: Anomaly()):
        self.state = inputState
        self.state_machine = inputResponder
        self.anomaly = inputAnomaly

    def run(self):
        startTime = time.time()
        while True:
            
            time.sleep(1 / self.frequency)
            self.step()

            print(self.state_machine.stepNum)
            print(self.state["0"])

            currTime = time.time()
            if currTime - startTime >= 5:
                break

    def step(self):
        self.state = self.anomaly.step(self.state,self.state_machine.stepNum)
