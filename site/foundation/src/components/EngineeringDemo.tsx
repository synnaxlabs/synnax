// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useId, useRef, useState } from "react";

import { DeferredScene } from "@/components/DeferredScene";
import { type EngineeringMode } from "@/components/EngineeringScene";

const loadScene = () =>
  import("@/components/EngineeringScene").then(({ EngineeringScene }) => ({
    default: EngineeringScene,
  }));

type Stage = 0 | 1 | 2 | 3 | 4;
interface Scenario {
  label: string;
  status: string;
  title: string;
  copy: string;
  notes: [string, string];
  evidence?: {
    quality: string;
    validation: string;
    delivery: string;
    buffer: string;
    gap: string;
  };
}

const scenarios: Record<EngineeringMode, Scenario[]> = {
  reliability: [
    {
      label: "Flowing",
      status: "ACQUIRED → CONFIRMED",
      title: "Know where every stream stands.",
      copy: "Follow telemetry from equipment and Ignition through Foundation to InfluxDB. Keep source quality, validation, and confirmed delivery visible as separate signals.",
      notes: [
        "Expect an update at least every second",
        "Confirm writes at the destination",
      ],
      evidence: {
        quality: "Good",
        validation: "Fresh · 180 ms old",
        delivery: "Confirmed · 210 ms lag",
        buffer: "0 samples pending",
        gap: "No recorded gaps",
      },
    },
    {
      label: "Source silent",
      status: "UPDATE EXPECTATION MISSED",
      title: "Connected does not mean current.",
      copy: "The Ignition connection stays online, but this stream stops updating. Foundation detects the missed expectation and identifies the last sample received.",
      notes: [
        "Expected every 1 s · last update 18 s ago",
        "Last reported quality stays attached to its sample",
      ],
      evidence: {
        quality: "Good · last sample",
        validation: "Stale · 18 s old",
        delivery: "Last sample confirmed",
        buffer: "0 samples pending",
        gap: "18 s without an update",
      },
    },
    {
      label: "Destination delayed",
      status: "ACQUIRING · DELIVERY BEHIND",
      title: "Find the delay. Keep acquiring.",
      copy: "Equipment keeps running and fresh samples reach Foundation. InfluxDB writes fall behind. Delivery lag and the local backlog expose where the data stopped.",
      notes: [
        "Local retention continues while delivery waits",
        "Source quality is unchanged by delivery lag",
      ],
      evidence: {
        quality: "Good",
        validation: "Fresh · 180 ms old",
        delivery: "Delayed · 42 s lag",
        buffer: "18,400 samples pending",
        gap: "Buffered · no confirmed loss",
      },
    },
    {
      label: "Recovering",
      status: "LIVE + RETAINED TELEMETRY",
      title: "See what recovers. See what cannot.",
      copy: "After a longer outage, retained telemetry catches up alongside live data. A four-second interval exceeded local retention. That gap remains visible as delivery recovers.",
      notes: [
        "14,200 of 18,400 retained samples delivered",
        "Expired commands are never replayed",
      ],
      evidence: {
        quality: "Good",
        validation: "Fresh · 180 ms old",
        delivery: "Catching up · 12 s lag",
        buffer: "4,200 samples pending",
        gap: "4 s gap · outside retention",
      },
    },
    {
      label: "Value check",
      status: "VALIDATION CONDITION",
      title: "Keep the reading. Add the context.",
      copy: "A pressure reading arrives on time, marked good by its source. It exceeds the configured 0–80 bar range. Foundation exposes that result alongside the original value and source quality.",
      notes: [
        "Observed 86.4 bar · expected 0–80 bar",
        "Validation does not rewrite source quality",
      ],
      evidence: {
        quality: "Good",
        validation: "86.4 bar · out of range",
        delivery: "Confirmed · 210 ms lag",
        buffer: "0 samples pending",
        gap: "No recorded gaps",
      },
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
  const root = useRef<HTMLDivElement>(null);
  const id = useId();

  useEffect(() => {
    const element = root.current;
    if (element == null) return;
    let visible = typeof IntersectionObserver === "undefined";
    const preference = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    const sync = (): void => {
      element.dataset.motion =
        visible && !document.hidden && !preference?.matches ? "running" : "paused";
    };
    const observer =
      typeof IntersectionObserver === "undefined"
        ? undefined
        : new IntersectionObserver(([entry]) => {
            visible = entry.isIntersecting;
            sync();
          });
    observer?.observe(element);
    preference?.addEventListener("change", sync);
    document.addEventListener("visibilitychange", sync);
    sync();
    return () => {
      observer?.disconnect();
      preference?.removeEventListener("change", sync);
      document.removeEventListener("visibilitychange", sync);
    };
  }, []);

  const selected = scenarios[mode][stage];
  return (
    <div
      id="data-health"
      className="engineering-demo"
      data-mode={mode}
      data-stage={stage}
      data-motion="paused"
      ref={root}
    >
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
        <div className="engineering-explanation">
          <p className="engineering-state" role="status" aria-atomic="true">
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
        </div>
        <div className="engineering-drawing">
          <div className="engineering-drawing-reference" aria-hidden="true">
            <span>
              {mode === "reliability" ? "ILLUSTRATIVE DATA PATH" : "FOUNDATION"}
            </span>
            <span>ORTHOGRAPHIC → {String(stage + 1).padStart(2, "0")}</span>
          </div>
          <DeferredScene
            load={loadScene}
            sceneProps={{ mode, stage }}
            viewBox="0 100 900 400"
          />
          {selected.evidence != null && (
            <p className="engineering-health-path" aria-hidden="true">
              Ignition <span>→</span> Foundation <span>→</span> InfluxDB
            </p>
          )}
          {selected.evidence != null && (
            <div className="engineering-health-readout">
              <dl className="engineering-health-signals">
                <div>
                  <dt>Source quality</dt>
                  <dd>{selected.evidence.quality}</dd>
                </div>
                <div data-attention={stage === 1 || stage === 4}>
                  <dt>Validation</dt>
                  <dd>{selected.evidence.validation}</dd>
                </div>
                <div data-attention={stage === 2}>
                  <dt>Delivery</dt>
                  <dd>{selected.evidence.delivery}</dd>
                </div>
              </dl>
              <div className="engineering-health-retention">
                <span>
                  <span className="engineering-health-key">BUFFER</span>
                  {selected.evidence.buffer}
                </span>
                <span data-gap={stage === 3}>{selected.evidence.gap}</span>
              </div>
            </div>
          )}
        </div>
      </div>
      <div className="engineering-scenario-bar">
        <span className="engineering-scenario-label">
          {mode === "reliability" ? "INSPECT A CONDITION" : "EXPLORE THE BEHAVIOR"}
        </span>
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
              data-secondary={mode === "reliability" && index === 4}
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
