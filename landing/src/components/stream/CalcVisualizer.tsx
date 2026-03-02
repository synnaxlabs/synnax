import {
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { CodePanel } from "@/components/shared/CodePanel";
import {
  CALC_EXAMPLES,
  ZERO_CALC_STATE,
} from "@/components/stream/calcTimeline";
import { Diagram } from "@/components/stream/diagrams";

interface CalcVisualizerProps {
  codeHtmls: string[];
}

export const CalcVisualizer = ({
  codeHtmls,
}: CalcVisualizerProps): ReactElement => {
  const [activeTab, setActiveTab] = useState(0);
  const [stepIndex, setStepIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const example = CALC_EXAMPLES[activeTab];
  const step = example.steps[stepIndex];
  const diagramState = { ...ZERO_CALC_STATE, ...step.state };

  const clearTimer = useCallback(() => {
    if (timeoutRef.current != null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (paused) return;
    clearTimer();
    timeoutRef.current = setTimeout(() => {
      setStepIndex((prev) => (prev + 1) % example.steps.length);
    }, step.duration);
    return clearTimer;
  }, [
    stepIndex,
    activeTab,
    paused,
    example.steps.length,
    step.duration,
    clearTimer,
  ]);

  const handleTabClick = useCallback(
    (index: number) => {
      if (index === activeTab) return;
      setActiveTab(index);
      setStepIndex(0);
    },
    [activeTab],
  );

  return (
    <div
      className="calc-visualizer viz-container"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="viz-tabs">
        {CALC_EXAMPLES.map((ex, i) => (
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
          <CodePanel
            html={codeHtmls[activeTab]}
            activeLines={step.activeLines}
          />
        </div>
        <div className="viz-diagram">
          <Diagram def={example.diagram} state={diagramState} />
        </div>
      </div>
      <div className="viz-progress">
        {example.steps.map((_, i) => (
          <div
            key={i}
            className={`viz-dot${i === stepIndex ? " viz-dot--active" : ""}`}
          />
        ))}
      </div>
    </div>
  );
};
