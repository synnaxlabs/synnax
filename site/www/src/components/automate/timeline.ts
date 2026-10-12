// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface DiagramState {
  stage: string;
  pressValve: boolean;
  ventValve: boolean;
  pressure: number;
  authority: number;
  alarmStatus: "ok" | "warning";
  activeNode: string;
}

export interface ExecutionStep {
  activeLines: number[];
  duration: number;
  state: Partial<DiagramState>;
}

export interface Example {
  id: string;
  title: string;
  /** The Arc source that the steps highlight, line by line. */
  code: string;
  steps: ExecutionStep[];
}

export const ZERO_DIAGRAM_STATE: DiagramState = {
  stage: "idle",
  pressValve: false,
  ventValve: false,
  pressure: 0,
  authority: 100,
  alarmStatus: "ok",
  activeNode: "",
};

export const EXAMPLES: Example[] = [
  {
    id: "pressure",
    title: "Pressurization sequence",
    code: `sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt > 500 => maintain
    }
    stage maintain {
        0 -> press_vlv_cmd,
        wait{duration=5s} => vent
    }
    stage vent {
        1 -> vent_vlv_cmd,
        press_pt < 10 => complete
    }
    stage complete {
        0 -> vent_vlv_cmd,
        0 -> press_vlv_cmd
    }
}`,
    steps: [
      // seq main
      {
        activeLines: [1],
        duration: 1200,
        state: { stage: "main" },
      },
      // stage press
      {
        activeLines: [2],
        duration: 1200,
        state: { stage: "press" },
      },
      // open press valve
      {
        activeLines: [3],
        duration: 1400,
        state: { stage: "press", pressValve: true, pressure: 250 },
      },
      // check condition (pressure rising toward 500)
      {
        activeLines: [4],
        duration: 1400,
        state: { stage: "press", pressValve: true, pressure: 520 },
      },
      // stage maintain (press valve still open from previous stage)
      {
        activeLines: [6],
        duration: 1200,
        state: { stage: "maintain", pressValve: true, pressure: 510 },
      },
      // close press valve
      {
        activeLines: [7],
        duration: 1200,
        state: { stage: "maintain", pressValve: false, pressure: 505 },
      },
      // wait
      {
        activeLines: [8],
        duration: 1800,
        state: { stage: "maintain", pressValve: false, pressure: 500 },
      },
      // stage vent
      {
        activeLines: [10],
        duration: 1200,
        state: { stage: "vent", pressure: 495 },
      },
      // open vent valve
      {
        activeLines: [11],
        duration: 1400,
        state: { stage: "vent", ventValve: true, pressure: 300 },
      },
      // check condition (pressure dropping toward 10)
      {
        activeLines: [12],
        duration: 1400,
        state: { stage: "vent", ventValve: true, pressure: 5 },
      },
      // stage complete
      {
        activeLines: [14],
        duration: 1200,
        state: { stage: "complete", ventValve: true, pressure: 5 },
      },
      // close vent, close press
      {
        activeLines: [15, 16],
        duration: 1800,
        state: {
          stage: "complete",
          pressValve: false,
          ventValve: false,
          pressure: 0,
        },
      },
    ],
  },
  {
    id: "alarm",
    title: "Alarm monitoring",
    code: `func check_pressure(p f64) u8 {
    return p > 750.0
}

press_pt -> check_pressure{}
    -> stable_for{duration=500ms}
    -> select{} -> {
        true: set_status{
            status_key="press_alarm",
            variant="warning",
            message="Pressure exceeds 750 PSI"
        },
        false: set_status{
            status_key="press_alarm",
            variant="success",
            message="Pressure nominal"
        }
    }`,
    steps: [
      // func definition
      {
        activeLines: [1, 2, 3],
        duration: 1400,
        state: { pressure: 400 },
      },
      // sensor reads 400 PSI
      {
        activeLines: [5],
        duration: 1200,
        state: { activeNode: "sensor", pressure: 400 },
      },
      // check_pressure evaluates (400 < 750)
      {
        activeLines: [2],
        duration: 1200,
        state: { activeNode: "check", pressure: 400 },
      },
      // stable_for 500ms
      {
        activeLines: [6],
        duration: 1200,
        state: { activeNode: "stable", pressure: 400 },
      },
      // select → false → nominal
      {
        activeLines: [13, 14, 15, 16],
        duration: 1800,
        state: { activeNode: "select-false", pressure: 400, alarmStatus: "ok" },
      },
      // sensor reads 760 PSI
      {
        activeLines: [5],
        duration: 1200,
        state: { activeNode: "sensor", pressure: 760 },
      },
      // check_pressure evaluates (760 > 750)
      {
        activeLines: [2],
        duration: 1200,
        state: { activeNode: "check", pressure: 760 },
      },
      // stable_for 500ms
      {
        activeLines: [6],
        duration: 1200,
        state: { activeNode: "stable", pressure: 760 },
      },
      // select → true → warning
      {
        activeLines: [8, 9, 10, 11],
        duration: 2000,
        state: {
          activeNode: "select-true",
          pressure: 760,
          alarmStatus: "warning",
        },
      },
    ],
  },
  {
    id: "abort",
    title: "Abort sequence",
    code: `authority (
    200
    press_vlv_cmd 100
    vent_vlv_cmd 100
)

start_cmd => main

sequence main {
    stage normal {
        1 -> press_vlv_cmd,
        press_pt > 800 => emergency
    }
    stage emergency {
        set_authority{value=255},
        0 -> press_vlv_cmd,
        1 -> vent_vlv_cmd,
        press_pt < 50 => safed
    }
    stage safed {
        0 -> press_vlv_cmd,
        0 -> vent_vlv_cmd
    }
}`,
    steps: [
      // authority declaration
      {
        activeLines: [1, 2, 3, 4, 5],
        duration: 1400,
        state: { stage: "idle", authority: 200 },
      },
      // start trigger
      {
        activeLines: [7],
        duration: 1200,
        state: { stage: "waiting", authority: 200 },
      },
      // stage normal
      {
        activeLines: [10],
        duration: 1200,
        state: { stage: "normal", authority: 200 },
      },
      // open press valve
      {
        activeLines: [11],
        duration: 1400,
        state: { stage: "normal", authority: 200, pressValve: true, pressure: 450 },
      },
      // condition met
      {
        activeLines: [12],
        duration: 1200,
        state: { stage: "normal", authority: 200, pressValve: true, pressure: 810 },
      },
      // enter emergency (press valve still open)
      {
        activeLines: [14],
        duration: 1200,
        state: { stage: "emergency", authority: 200, pressValve: true, pressure: 810 },
      },
      // set authority (press valve still open)
      {
        activeLines: [15],
        duration: 1200,
        state: { stage: "emergency", authority: 255, pressValve: true, pressure: 800 },
      },
      // close press valve
      {
        activeLines: [16],
        duration: 1200,
        state: { stage: "emergency", authority: 255, pressure: 790 },
      },
      // open vent valve
      {
        activeLines: [17],
        duration: 1400,
        state: { stage: "emergency", authority: 255, ventValve: true, pressure: 400 },
      },
      // condition met
      {
        activeLines: [18],
        duration: 1400,
        state: { stage: "emergency", authority: 255, ventValve: true, pressure: 30 },
      },
      // enter safed (vent valve still open)
      {
        activeLines: [20],
        duration: 1200,
        state: { stage: "safed", authority: 255, ventValve: true, pressure: 20 },
      },
      // close both valves
      {
        activeLines: [21, 22],
        duration: 1800,
        state: {
          stage: "safed",
          authority: 255,
          pressValve: false,
          ventValve: false,
          pressure: 0,
        },
      },
    ],
  },
];
