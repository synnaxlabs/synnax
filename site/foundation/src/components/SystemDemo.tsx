// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useState } from "react";

import { DeferredScene } from "@/components/DeferredScene";
import { type SystemLayer } from "@/components/IndustrialScene";

const loadScene = () =>
  import("@/components/IndustrialScene").then(({ IndustrialScene }) => ({
    default: IndustrialScene,
  }));

const layers: {
  id: SystemLayer;
  label: string;
  title: string;
  copy: string;
  detail: string;
}[] = [
  {
    id: "operation",
    label: "Equipment",
    title: "Start with the real world.",
    copy: "The PLC on the production line. The instruments on the test stand. The LabVIEW application or SCADA system already running your facility.",
    detail:
      "Foundation nodes stream data and carry supervisory commands to and from your existing systems.",
  },
  {
    id: "connectivity",
    label: "Cluster",
    title: "Form the cluster.",
    copy: "A test stand in one building. A production line in another. A remote site across the country.",
    detail:
      "Foundation nodes communicate within and across sites, bringing these systems into one cluster.",
  },
  {
    id: "applications",
    label: "Telemetry",
    title: "Build a real-time picture.",
    copy: "Link unified telemetry streams to databases, analytics, orchestration platforms, and machine intelligence.",
    detail:
      "Give every application and agent the same live context from your operation.",
  },
  {
    id: "control",
    label: "Control",
    title: "Act on the physical world.",
    copy: "Change a setpoint from your application. Update a calibration from a centralized system. Receive an acknowledgment.",
    detail:
      "Foundation routes supervisory commands to any piece of your infrastructure. It protects critical systems through sophisticated rules, interlocks, and access control mechanisms.",
  },
];

export const SystemDemo = (): ReactElement => {
  const [layer, setLayer] = useState<SystemLayer>("operation");
  const activeIndex = layers.findIndex(({ id }) => id === layer);
  const activeLayer = layers[activeIndex];

  return (
    <div className="system-demo" data-layer={layer}>
      <div className="world-stage">
        <div className="world-toolbar">
          <div
            className="world-layer-controls"
            role="group"
            aria-label="How Foundation works"
          >
            {layers.map(({ id, label }, i) => (
              <button
                type="button"
                key={id}
                aria-pressed={layer === id}
                aria-controls="diagram-explanation"
                onClick={() => setLayer(id)}
              >
                <span className="layer-number">0{i + 1}</span>
                {label}
              </button>
            ))}
          </div>
          <div className="world-readout">
            <svg className="world-axes" viewBox="0 0 88 80" aria-hidden="true">
              <g className="scene-coordinate" transform="translate(40 49)">
                <path d="M0 0 L26 11 M0 0 L-20 8 M0 0 V-28" />
                <text x={31} y={17}>
                  X
                </text>
                <text x={-31} y={15}>
                  Y
                </text>
                <text x={-3} y={-34}>
                  Z
                </text>
              </g>
            </svg>
            <span className="world-view">SYSTEM VIEW → 0{activeIndex + 1}</span>
            <div className="world-status" role="status">
              <span className="network-dot" aria-hidden="true" />
              <span>All systems connected</span>
            </div>
          </div>
        </div>
        <div className="world-viewport">
          <DeferredScene
            load={loadScene}
            sceneProps={{ layer }}
            viewBox="0 0 1500 890"
          />
        </div>
        {layer === "control" && (
          <div
            className="world-flow-key"
            role="group"
            aria-label="Control flow directions"
          >
            <span>
              <svg className="world-command-key" viewBox="0 0 30 12" aria-hidden="true">
                <path d="M2 6H27M22 1L27 6L22 11" />
              </svg>
              Command to equipment
            </span>
            <span>
              <svg
                className="world-acknowledgment-key"
                viewBox="0 0 30 12"
                aria-hidden="true"
              >
                <path d="M28 6H3M8 1L3 6L8 11" />
              </svg>
              Acknowledgment to software
            </span>
          </div>
        )}
        <article
          id="diagram-explanation"
          className="world-caption"
          aria-labelledby="diagram-explanation-title"
        >
          <div className="chapter-copy" aria-live="polite" aria-atomic="true">
            <span className="chapter-topline">
              0{activeIndex + 1} of 0{layers.length} · {activeLayer.label}
            </span>
            <h3 id="diagram-explanation-title">{activeLayer.title}</h3>
            <p>{activeLayer.copy}</p>
            <p className="chapter-detail">{activeLayer.detail}</p>
          </div>
          <button
            type="button"
            className="chapter-next"
            aria-controls="diagram-explanation"
            onClick={() => setLayer(layers[(activeIndex + 1) % layers.length].id)}
          >
            Next step <Icon.Arrow.Right aria-hidden="true" />
          </button>
        </article>
      </div>
    </div>
  );
};
