// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CodePanel } from "@/components/common/CodePanel";
import { useTimeline } from "@/components/common/useTimeline";
import { CALC_EXAMPLES, ZERO_CALC_STATE } from "@/components/stream/calcTimeline";
import { Diagram } from "@/components/stream/diagrams";

interface CalcVisualizerProps {
  codeHtmls: string[];
}

export const CalcVisualizer = ({ codeHtmls }: CalcVisualizerProps): ReactElement => {
  const { activeTab, stepIndex, step, playKey, selectTab, containerProps } =
    useTimeline(CALC_EXAMPLES);

  const example = CALC_EXAMPLES[activeTab];
  const diagramState = { ...ZERO_CALC_STATE, ...step.state };

  return (
    <div className="calc-visualizer viz-container" {...containerProps}>
      <div className="viz-tabs">
        {CALC_EXAMPLES.map((ex, i) => (
          <button
            key={ex.id}
            className={`viz-tab${i === activeTab ? " viz-tab--active" : ""}`}
            onClick={() => selectTab(i)}
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
          <Diagram def={example.diagram} state={diagramState} />
        </div>
      </div>
      <div className="viz-progress">
        {example.steps.map((s, i) => (
          <div
            key={i === stepIndex ? playKey : i}
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
