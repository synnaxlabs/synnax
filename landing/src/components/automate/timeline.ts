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
    title: "Pressure Sequence",
    steps: [
      {
        activeLines: [1],
        duration: 1200,
        state: { stage: "main" },
      },
      {
        activeLines: [2, 3],
        duration: 1400,
        state: { stage: "press", pressValve: true, pressure: 120 },
      },
      {
        activeLines: [3, 4],
        duration: 1200,
        state: { stage: "press", pressValve: true, pressure: 340 },
      },
      {
        activeLines: [4],
        duration: 1000,
        state: { stage: "press", pressValve: true, pressure: 520 },
      },
      {
        activeLines: [6, 7, 8],
        duration: 1400,
        state: { stage: "maintain", pressValve: false, pressure: 510 },
      },
      {
        activeLines: [8],
        duration: 1800,
        state: { stage: "maintain", pressValve: false, pressure: 505 },
      },
      {
        activeLines: [10, 11],
        duration: 1400,
        state: { stage: "vent", ventValve: true, pressure: 380 },
      },
      {
        activeLines: [11, 12],
        duration: 1200,
        state: { stage: "vent", ventValve: true, pressure: 150 },
      },
      {
        activeLines: [12],
        duration: 1000,
        state: { stage: "vent", ventValve: true, pressure: 5 },
      },
      {
        activeLines: [14, 15, 16],
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
    title: "Alarm Monitoring",
    steps: [
      {
        activeLines: [1, 2, 3],
        duration: 1400,
        state: { activeNode: "func", pressure: 400 },
      },
      {
        activeLines: [5],
        duration: 1200,
        state: { activeNode: "sensor", pressure: 400 },
      },
      {
        activeLines: [5, 6],
        duration: 1200,
        state: { activeNode: "check", pressure: 400 },
      },
      {
        activeLines: [6, 7],
        duration: 1000,
        state: { activeNode: "stable", pressure: 400 },
      },
      {
        activeLines: [7, 8, 11, 12, 13],
        duration: 1800,
        state: { activeNode: "select-false", pressure: 400, alarmStatus: "ok" },
      },
      {
        activeLines: [5],
        duration: 1200,
        state: { activeNode: "sensor", pressure: 760 },
      },
      {
        activeLines: [5, 6],
        duration: 1200,
        state: { activeNode: "check", pressure: 760 },
      },
      {
        activeLines: [6, 7],
        duration: 1000,
        state: { activeNode: "stable", pressure: 760 },
      },
      {
        activeLines: [7, 8, 9, 10],
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
    title: "Abort Sequence",
    steps: [
      {
        activeLines: [1, 2, 3, 4],
        duration: 1400,
        state: { stage: "authority", authority: 200, pressure: 200 },
      },
      {
        activeLines: [6],
        duration: 1000,
        state: { stage: "waiting", authority: 200, pressure: 200 },
      },
      {
        activeLines: [8, 9, 10],
        duration: 1400,
        state: {
          stage: "normal",
          authority: 200,
          pressValve: true,
          pressure: 450,
        },
      },
      {
        activeLines: [10],
        duration: 1200,
        state: {
          stage: "normal",
          authority: 200,
          pressValve: true,
          pressure: 650,
        },
      },
      {
        activeLines: [10, 11],
        duration: 1000,
        state: {
          stage: "normal",
          authority: 200,
          pressValve: true,
          pressure: 810,
        },
      },
      {
        activeLines: [12, 13],
        duration: 1400,
        state: { stage: "emergency", authority: 255, pressure: 810 },
      },
      {
        activeLines: [14, 15],
        duration: 1400,
        state: {
          stage: "emergency",
          authority: 255,
          pressValve: false,
          ventValve: true,
          pressure: 500,
        },
      },
      {
        activeLines: [15, 16],
        duration: 1200,
        state: {
          stage: "emergency",
          authority: 255,
          ventValve: true,
          pressure: 200,
        },
      },
      {
        activeLines: [16],
        duration: 1000,
        state: {
          stage: "emergency",
          authority: 255,
          ventValve: true,
          pressure: 30,
        },
      },
      {
        activeLines: [18, 19, 20],
        duration: 2000,
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
