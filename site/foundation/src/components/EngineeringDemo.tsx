// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

import { type ReactElement, useId, useState } from "react";

import { DeferredScene } from "@/components/DeferredScene";
import { type EngineeringMode } from "@/components/EngineeringScene";

const loadScene = () =>
  import("@/components/EngineeringScene").then(({ EngineeringScene }) => ({
    default: EngineeringScene,
  }));

type Stage = 0 | 1 | 2;
interface Scenario {
  label: string;
  status: string;
  title: string;
  copy: string;
  notes: [string, string];
}

const scenarios: Record<EngineeringMode, [Scenario, Scenario, Scenario]> = {
  reliability: [
    {
      label: "Connected",
      status: "LINK AVAILABLE",
      title: "Durability starts at the source.",
      copy: "Nodes retain telemetry locally and track delivery across the network. Your equipment and your destinations can operate independently.",
      notes: ["Durable, bounded local buffers", "Explicit delivery progress"],
    },
    {
      label: "Interrupt the link",
      status: "LOCAL BUFFERING",
      title: "A lost link is a handled state.",
      copy: "Acquisition continues into the local buffer while remote delivery pauses. Retention and disk capacity define how much data the node can hold.",
      notes: ["Telemetry retained at the node", "Command deadlines still apply"],
    },
    {
      label: "Restore the link",
      status: "DELIVERY RESUMED",
      title: "Catch up with the operation.",
      copy: "Forward retained telemetry when the connection returns, with priority for live data. Expired commands stay expired rather than replaying against equipment.",
      notes: ["Live data ahead of backfill", "No replay of expired commands"],
    },
  ],
  performance: [
    {
      label: "Ingest",
      status: "VALIDATE AT INGEST",
      title: "Preserve the signal. Drop the overhead.",
      copy: "Bring device data into a common channel representation at ingress. Carry values, timestamps, and quality through the same data path.",
      notes: ["Validation at the boundary", "Consistent channel representation"],
    },
    {
      label: "Move through memory",
      status: "BATCHED DATA PATH",
      title: "Every copy has a cost.",
      copy: "Group samples into frames and keep the hot path direct. Memory layout, batching, and ownership minimize copies, locks, and allocations.",
      notes: ["Batch samples into frames", "Minimize work between boundaries"],
    },
    {
      label: "Deliver",
      status: "QUIC TRANSPORT",
      title: "Keep live traffic moving.",
      copy: "Move frames between nodes over QUIC. Delivery priorities keep current telemetry moving while retained data catches up in the background.",
      notes: ["Multiplexed transport", "Explicit live and backfill priorities"],
    },
  ],
  deployment: [
    {
      label: "One node",
      status: "ONE BINARY",
      title: "Start where the equipment is.",
      copy: "Run Foundation on the machine beside your equipment. The same deployable binary model carries from a local host to a site or cloud environment.",
      notes: ["Linux, macOS, and Windows", "No separate management service"],
    },
    {
      label: "Across sites",
      status: "ONE MESH",
      title: "Add nodes. Keep one system.",
      copy: "Join hosts and sites into the same mesh. Nodes use direct connections and relay paths to work across different network environments.",
      notes: ["Devices, hosts, sites, and cloud", "Direct and relay connectivity"],
    },
    {
      label: "Desired state",
      status: "CONFIGURED AS CODE",
      title: "Make changes you can review.",
      copy: "Describe desired state in readable configuration files. Keep them in Git, inspect the changes, and apply the same model across your deployment.",
      notes: ["Versioned configuration", "Plan and apply workflows"],
    },
  ],
};

const modes: { id: EngineeringMode; label: string; number: string }[] = [
  { id: "reliability", label: "Reliability", number: "01" },
  { id: "performance", label: "Performance", number: "02" },
  { id: "deployment", label: "Deployment", number: "03" },
];

export const EngineeringDemo = (): ReactElement => {
  const [mode, setMode] = useState<EngineeringMode>("reliability");
  const [stage, setStage] = useState<Stage>(0);
  const id = useId();
  const selected = scenarios[mode][stage];
  return (
    <div className="engineering-demo" data-mode={mode} data-stage={stage}>
      <div className="engineering-demo-toolbar">
        <div
          className="engineering-modes"
          role="group"
          aria-label="Engineering attributes"
        >
          {modes.map(({ id: next, label, number }) => (
            <button
              key={next}
              type="button"
              aria-pressed={mode === next}
              aria-controls={`${id}-view`}
              onClick={() => {
                setMode(next);
                setStage(0);
              }}
            >
              <span className="engineering-control-index" aria-hidden="true">
                {number}
              </span>
              {label}
            </button>
          ))}
        </div>
        <span className="engineering-view-label" aria-hidden="true">
          SYSTEM BEHAVIOR
        </span>
      </div>
      <div className="engineering-demo-view" id={`${id}-view`}>
        <div className="engineering-explanation" aria-live="polite" aria-atomic="true">
          <p className="engineering-state">
            <span />
            {selected.status}
          </p>
          <h3>{selected.title}</h3>
          <p className="engineering-explanation-copy">{selected.copy}</p>
          <ul>
            {selected.notes.map((note) => (
              <li key={note}>{note}</li>
            ))}
          </ul>
          {mode === "reliability" && (
            <a
              href="https://docs.synnaxlabs.com/reliability/"
              className="engineering-evidence"
            >
              How Synnax tests reliability <span aria-hidden="true">↗</span>
            </a>
          )}
        </div>
        <div className="engineering-drawing">
          <div className="engineering-drawing-reference" aria-hidden="true">
            <span>FOUNDATION</span>
            <span>ORTHOGRAPHIC → {String(stage + 1).padStart(2, "0")}</span>
          </div>
          <DeferredScene
            load={loadScene}
            sceneProps={{ mode, stage }}
            viewBox="0 100 900 400"
          />
        </div>
      </div>
      <div className="engineering-scenario-bar">
        <span className="engineering-scenario-label">EXPLORE THE BEHAVIOR</span>
        <div
          role="group"
          aria-label={`${mode} scenarios`}
          className="engineering-scenarios"
        >
          {scenarios[mode].map(({ label }, index) => (
            <button
              key={`${mode}-${label}`}
              type="button"
              aria-pressed={stage === index}
              aria-controls={`${id}-view`}
              onClick={() => setStage(index as Stage)}
            >
              <span className="engineering-control-index" aria-hidden="true">
                {String(index + 1).padStart(2, "0")}
              </span>
              {label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};
