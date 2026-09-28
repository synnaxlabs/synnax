// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";

import { AbortDiagram } from "@/components/automate/diagrams/AbortDiagram";
import { PressureDiagram } from "@/components/automate/diagrams/PressureDiagram";
import {
  type DiagramState,
  EXAMPLES,
  ZERO_DIAGRAM_STATE,
} from "@/components/automate/timeline";
import { CodePanel } from "@/components/shared/CodePanel";
import type { CalcDiagramState } from "@/components/stream/calcTimeline";
import { AUTOMATE_ALARM_DIAGRAM, Diagram } from "@/components/stream/diagrams";

interface AutomateVisualizerProps {
  codeHtmls: string[];
}

const DIAGRAMS: Record<string, React.FC<{ state: DiagramState }>> = {
  pressure: PressureDiagram,
  abort: AbortDiagram,
};

const PID_WRAPPER_STYLE: React.CSSProperties = {
  position: "relative",
  width: "100%",
  height: "100%",
  overflow: "hidden",
  backgroundColor: "var(--pluto-gray-l2-30)",
  backgroundImage:
    "radial-gradient(circle, var(--pluto-gray-l5) 0.5px, transparent 0.5px)",
  backgroundSize: "16px 16px",
};

const alarmToCalcState = (state: DiagramState): CalcDiagramState => {
  const activeNodes: string[] = [];
  const nodeValues: Record<string, string> = {};
  const alarmNodes: string[] = [];
  const over = state.pressure > 750;

  nodeValues.sensor = `${state.pressure} PSI`;

  switch (state.activeNode) {
    case "sensor":
      activeNodes.push("sensor");
      break;
    case "check":
      activeNodes.push("check");
      nodeValues.check = over ? "> 750" : "< 750";
      break;
    case "stable":
      activeNodes.push("stable");
      nodeValues.check = over ? "> 750" : "< 750";
      nodeValues.stable = "500ms";
      break;
    case "select-false":
      activeNodes.push("nominal");
      nodeValues.check = "< 750";
      nodeValues.stable = "500ms";
      nodeValues.nominal = "nominal";
      break;
    case "select-true":
      activeNodes.push("warning");
      nodeValues.check = "> 750";
      nodeValues.stable = "500ms";
      nodeValues.warning = "warning";
      alarmNodes.push("warning");
      break;
  }

  return { activeNodes, nodeValues, excludedNodes: [], alarmNodes };
};

export const AutomateVisualizer = ({
  codeHtmls,
}: AutomateVisualizerProps): ReactElement => {
  const [activeTab, setActiveTab] = useState(0);
  const [stepIndex, setStepIndex] = useState(0);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const example = EXAMPLES[activeTab];
  const step = example.steps[stepIndex];
  const diagramState: DiagramState = { ...ZERO_DIAGRAM_STATE, ...step.state };
  const isAlarm = example.id === "alarm";
  const DiagramComponent = DIAGRAMS[example.id];

  const clearTimer = useCallback(() => {
    if (timeoutRef.current != null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    clearTimer();
    timeoutRef.current = setTimeout(() => {
      setStepIndex((prev) => (prev + 1) % example.steps.length);
    }, step.duration);
    return clearTimer;
  }, [stepIndex, activeTab, example.steps.length, step.duration, clearTimer]);

  const handleTabClick = useCallback(
    (index: number) => {
      if (index === activeTab) return;
      setActiveTab(index);
      setStepIndex(0);
    },
    [activeTab],
  );

  return (
    <div className="viz-container">
      <div className="viz-tabs">
        {EXAMPLES.map((ex, i) => (
          <button
            key={ex.id}
            className={`viz-tab${i === activeTab ? " viz-tab--active" : ""}`}
            onClick={() => handleTabClick(i)}
          >
            {ex.title}
          </button>
        ))}
      </div>
      <div className="viz-content">
        <div className="viz-code">
          <CodePanel html={codeHtmls[activeTab]} activeLines={step.activeLines} />
        </div>
        <div className="viz-diagram">
          {isAlarm ? (
            <Diagram
              def={AUTOMATE_ALARM_DIAGRAM}
              state={alarmToCalcState(diagramState)}
            />
          ) : (
            DiagramComponent != null && (
              <div style={PID_WRAPPER_STYLE}>
                <DiagramComponent state={diagramState} />
              </div>
            )
          )}
        </div>
      </div>
      <div className="viz-progress">
        {example.steps.map((s, i) => (
          <div
            key={i === stepIndex ? `${activeTab}-${stepIndex}` : i}
            className={`viz-dot${i === stepIndex ? " viz-dot--active" : ""}`}
            style={
              i === stepIndex
                ? ({ "--step-duration": `${s.duration}ms` } as React.CSSProperties)
                : undefined
            }
          />
        ))}
      </div>
    </div>
  );
};
