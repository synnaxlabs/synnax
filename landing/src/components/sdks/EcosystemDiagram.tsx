import { Logo } from "@synnaxlabs/media";
import { Icon } from "@synnaxlabs/pluto";
import type { CSSProperties, ReactElement } from "react";

import type { CalcDiagramState } from "@/components/stream/calcTimeline";
import { Diagram } from "@/components/stream/diagrams";
import type { DiagramDef } from "@/components/stream/diagrams";

const SynnaxIcon = ({ style }: { style?: CSSProperties }): ReactElement => (
  <Logo
    variant="icon"
    style={{ ...style, "--logo-color": "black" } as CSSProperties}
  />
);

const DIAGRAM: DiagramDef = {
  viewBox: "-100 -30 1100 360",
  variant: "pill",
  nodes: [
    {
      id: "hardware_daq",
      x: 110,
      y: 40,
      w: 130,
      h: 50,
      label: "hardware_daq",
      icon: Icon.Logo.NI,
    },
    {
      id: "ext_ecu",
      x: 110,
      y: 150,
      w: 130,
      h: 50,
      label: "ext_ecu",
      icon: Icon.CPlusPlus,
    },
    {
      id: "custom_io",
      x: 110,
      y: 260,
      w: 130,
      h: 50,
      label: "custom_io",
      icon: Icon.Logo.Modbus,
    },
    {
      id: "synnax",
      x: 450,
      y: 150,
      w: 76,
      h: 76,
      label: "synnax",
      icon: SynnaxIcon,
    },
    {
      id: "ml_pipeline",
      x: 790,
      y: 40,
      w: 130,
      h: 50,
      label: "ml_pipeline",
      icon: Icon.Python,
    },
    {
      id: "dashboard",
      x: 790,
      y: 150,
      w: 130,
      h: 50,
      label: "dashboard",
      icon: Icon.TypeScript,
    },
    {
      id: "test_runner",
      x: 790,
      y: 260,
      w: 130,
      h: 50,
      label: "test_runner",
      icon: Icon.CPlusPlus,
    },
  ],
  edges: [
    { from: "hardware_daq", to: "synnax" },
    { from: "ext_ecu", to: "synnax" },
    { from: "custom_io", to: "synnax" },
    { from: "synnax", to: "ml_pipeline" },
    { from: "synnax", to: "dashboard" },
    { from: "synnax", to: "test_runner" },
  ],
};

const STATE: CalcDiagramState = {
  activeNodes: ["synnax"],
  nodeValues: {
    hardware_daq: "NI · LabJack",
    ext_ecu: "C++ embedded",
    custom_io: "Modbus · serial",
    synnax: "",
    ml_pipeline: "Python",
    dashboard: "TypeScript",
    test_runner: "CI/CD",
  },
  excludedNodes: [],
  alarmNodes: [],
};

export const EcosystemDiagram = (): ReactElement => (
  <div className="sdks-ecosystem">
    <Diagram def={DIAGRAM} state={STATE} />
  </div>
);
