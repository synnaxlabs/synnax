import {
  ALARM_DIAGRAM,
  CONVERSION_DIAGRAM,
  FFT_DIAGRAM,
  MASSFLOW_DIAGRAM,
  MIXTURE_DIAGRAM,
  VOTING_DIAGRAM,
} from "@/components/stream/diagrams";
import type { DiagramDef } from "@/components/stream/diagrams";

export interface CalcDiagramState {
  activeNodes: string[];
  nodeValues: Record<string, string>;
  excludedNodes: string[];
  alarmNodes: string[];
}

export interface CalcStep {
  activeLines: number[];
  duration: number;
  state: Partial<CalcDiagramState>;
}

export interface CalcExample {
  id: string;
  title: string;
  diagram: DiagramDef;
  steps: CalcStep[];
}

export const ZERO_CALC_STATE: CalcDiagramState = {
  activeNodes: [],
  nodeValues: {},
  excludedNodes: [],
  alarmNodes: [],
};

export const CALC_EXAMPLES: CalcExample[] = [
  {
    id: "conversion",
    title: "Conversion",
    diagram: CONVERSION_DIAGRAM,
    steps: [
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["celsius"],
          nodeValues: { celsius: "22.3 °C" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["compute"],
          nodeValues: { celsius: "22.3 °C" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["fahrenheit"],
          nodeValues: { celsius: "22.3 °C", fahrenheit: "72.1 °F" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["celsius"],
          nodeValues: { celsius: "23.1 °C", fahrenheit: "72.1 °F" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["compute"],
          nodeValues: { celsius: "23.1 °C", fahrenheit: "72.1 °F" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["fahrenheit"],
          nodeValues: { celsius: "23.1 °C", fahrenheit: "73.6 °F" },
        },
      },
    ],
  },
  {
    id: "mixture",
    title: "Mixture Ratio",
    diagram: MIXTURE_DIAGRAM,
    steps: [
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["ox", "fuel"],
          nodeValues: { ox: "2.45 kg/s", fuel: "1.12 kg/s" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["compute"],
          nodeValues: { ox: "2.45 kg/s", fuel: "1.12 kg/s" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["ratio"],
          nodeValues: { ox: "2.45 kg/s", fuel: "1.12 kg/s", ratio: "2.19" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["ox", "fuel"],
          nodeValues: { ox: "2.51 kg/s", fuel: "1.09 kg/s", ratio: "2.19" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["compute"],
          nodeValues: { ox: "2.51 kg/s", fuel: "1.09 kg/s", ratio: "2.19" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["ratio"],
          nodeValues: { ox: "2.51 kg/s", fuel: "1.09 kg/s", ratio: "2.30" },
        },
      },
    ],
  },
  {
    id: "alarm",
    title: "Alarm",
    diagram: ALARM_DIAGRAM,
    steps: [
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["pressure"],
          nodeValues: { pressure: "680 PSI" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["decision"],
          nodeValues: { pressure: "680 PSI" },
        },
      },
      {
        activeLines: [4],
        duration: 1200,
        state: {
          activeNodes: ["alarm_off"],
          nodeValues: { pressure: "680 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["pressure"],
          nodeValues: { pressure: "720 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["decision"],
          nodeValues: { pressure: "720 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [4],
        duration: 1200,
        state: {
          activeNodes: ["alarm_off"],
          nodeValues: { pressure: "720 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["pressure"],
          nodeValues: { pressure: "785 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [1],
        duration: 1600,
        state: {
          activeNodes: ["decision"],
          nodeValues: { pressure: "785 PSI", alarm_off: "0" },
        },
      },
      {
        activeLines: [1, 2],
        duration: 1800,
        state: {
          activeNodes: ["alarm_on"],
          nodeValues: { pressure: "785 PSI", alarm_on: "1" },
          alarmNodes: ["alarm_on"],
        },
      },
    ],
  },
  {
    id: "massflow",
    title: "Mass Flow",
    diagram: MASSFLOW_DIAGRAM,
    steps: [
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["upstream", "downstream", "ox_temp"],
          nodeValues: {
            upstream: "310.2 kPa",
            downstream: "285.7 kPa",
            ox_temp: "-183.0 °C",
          },
        },
      },
      {
        activeLines: [2],
        duration: 1600,
        state: {
          activeNodes: ["density"],
          nodeValues: {
            upstream: "310.2 kPa",
            downstream: "285.7 kPa",
            ox_temp: "-183.0 °C",
            density: "ρ = 1141",
          },
        },
      },
      {
        activeLines: [3],
        duration: 1600,
        state: {
          activeNodes: ["orifice"],
          nodeValues: {
            upstream: "310.2 kPa",
            downstream: "285.7 kPa",
            ox_temp: "-183.0 °C",
            density: "ρ = 1141",
          },
        },
      },
      {
        activeLines: [3],
        duration: 1200,
        state: {
          activeNodes: ["mass_flow"],
          nodeValues: {
            upstream: "310.2 kPa",
            downstream: "285.7 kPa",
            ox_temp: "-183.0 °C",
            density: "ρ = 1141",
            mass_flow: "2.47 kg/s",
          },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["upstream", "downstream", "ox_temp"],
          nodeValues: {
            upstream: "312.8 kPa",
            downstream: "283.1 kPa",
            ox_temp: "-182.5 °C",
            density: "ρ = 1141",
            mass_flow: "2.47 kg/s",
          },
        },
      },
      {
        activeLines: [2],
        duration: 1600,
        state: {
          activeNodes: ["density"],
          nodeValues: {
            upstream: "312.8 kPa",
            downstream: "283.1 kPa",
            ox_temp: "-182.5 °C",
            density: "ρ = 1140",
            mass_flow: "2.47 kg/s",
          },
        },
      },
      {
        activeLines: [3],
        duration: 1600,
        state: {
          activeNodes: ["orifice"],
          nodeValues: {
            upstream: "312.8 kPa",
            downstream: "283.1 kPa",
            ox_temp: "-182.5 °C",
            density: "ρ = 1140",
            mass_flow: "2.47 kg/s",
          },
        },
      },
      {
        activeLines: [3],
        duration: 1200,
        state: {
          activeNodes: ["mass_flow"],
          nodeValues: {
            upstream: "312.8 kPa",
            downstream: "283.1 kPa",
            ox_temp: "-182.5 °C",
            density: "ρ = 1140",
            mass_flow: "2.61 kg/s",
          },
        },
      },
    ],
  },
  {
    id: "voting",
    title: "Sensor Voting",
    diagram: VOTING_DIAGRAM,
    steps: [
      {
        activeLines: [1, 2, 3],
        duration: 1400,
        state: {
          activeNodes: ["p1", "p2", "p3"],
          nodeValues: { p1: "748.2", p2: "749.1", p3: "748.7" },
        },
      },
      {
        activeLines: [5, 6],
        duration: 1400,
        state: {
          activeNodes: ["vote"],
          nodeValues: {
            p1: "748.2",
            p2: "749.1",
            p3: "748.7",
            vote: "3 of 3",
          },
        },
      },
      {
        activeLines: [6],
        duration: 1200,
        state: {
          activeNodes: ["trusted"],
          nodeValues: {
            p1: "748.2",
            p2: "749.1",
            p3: "748.7",
            vote: "3 of 3",
            trusted: "748.67",
          },
        },
      },
      {
        activeLines: [1, 2, 3],
        duration: 1400,
        state: {
          activeNodes: ["p1", "p2", "p3"],
          nodeValues: {
            p1: "751.3",
            p2: "750.8",
            p3: "751.1",
            vote: "3 of 3",
            trusted: "748.67",
          },
        },
      },
      {
        activeLines: [5, 6],
        duration: 1400,
        state: {
          activeNodes: ["vote"],
          nodeValues: {
            p1: "751.3",
            p2: "750.8",
            p3: "751.1",
            vote: "3 of 3",
            trusted: "748.67",
          },
        },
      },
      {
        activeLines: [6],
        duration: 1200,
        state: {
          activeNodes: ["trusted"],
          nodeValues: {
            p1: "751.3",
            p2: "750.8",
            p3: "751.1",
            vote: "3 of 3",
            trusted: "751.07",
          },
        },
      },
      {
        activeLines: [1, 2, 3],
        duration: 1400,
        state: {
          activeNodes: ["p1", "p2", "p3"],
          nodeValues: {
            p1: "749.5",
            p2: "749.2",
            p3: "782.1",
            vote: "3 of 3",
            trusted: "751.07",
          },
        },
      },
      {
        activeLines: [3, 8],
        duration: 1800,
        state: {
          activeNodes: ["vote"],
          nodeValues: {
            p1: "749.5",
            p2: "749.2",
            p3: "782.1",
            vote: "2 of 3",
            trusted: "751.07",
          },
          excludedNodes: ["p3"],
        },
      },
      {
        activeLines: [8],
        duration: 1400,
        state: {
          activeNodes: ["trusted"],
          nodeValues: {
            p1: "749.5",
            p2: "749.2",
            p3: "782.1",
            vote: "2 of 3",
            trusted: "749.35",
          },
          excludedNodes: ["p3"],
        },
      },
    ],
  },
  {
    id: "fft",
    title: "FFT Analysis",
    diagram: FFT_DIAGRAM,
    steps: [
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["accel"],
          nodeValues: { accel: "± 3.2 g" },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["window"],
          nodeValues: { accel: "± 3.2 g", window: "1024 pts" },
        },
      },
      {
        activeLines: [2],
        duration: 1600,
        state: {
          activeNodes: ["fft"],
          nodeValues: { accel: "± 3.2 g", window: "1024 pts" },
        },
      },
      {
        activeLines: [3],
        duration: 1400,
        state: {
          activeNodes: ["freqs"],
          nodeValues: { accel: "± 3.2 g", window: "1024 pts", freqs: "0–5 kHz" },
        },
      },
      {
        activeLines: [4, 5],
        duration: 1400,
        state: {
          activeNodes: ["freq"],
          nodeValues: {
            accel: "± 3.2 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "847 Hz",
          },
        },
      },
      {
        activeLines: [1],
        duration: 1200,
        state: {
          activeNodes: ["accel"],
          nodeValues: {
            accel: "± 2.8 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "847 Hz",
          },
        },
      },
      {
        activeLines: [1],
        duration: 1400,
        state: {
          activeNodes: ["window"],
          nodeValues: {
            accel: "± 2.8 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "847 Hz",
          },
        },
      },
      {
        activeLines: [2],
        duration: 1600,
        state: {
          activeNodes: ["fft"],
          nodeValues: {
            accel: "± 2.8 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "847 Hz",
          },
        },
      },
      {
        activeLines: [3],
        duration: 1400,
        state: {
          activeNodes: ["freqs"],
          nodeValues: {
            accel: "± 2.8 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "847 Hz",
          },
        },
      },
      {
        activeLines: [4, 5],
        duration: 1400,
        state: {
          activeNodes: ["freq"],
          nodeValues: {
            accel: "± 2.8 g",
            window: "1024 pts",
            freqs: "0–5 kHz",
            freq: "851 Hz",
          },
        },
      },
    ],
  },
];
