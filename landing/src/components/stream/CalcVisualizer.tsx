import {
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { CalcCodePanel } from "@/components/stream/CalcCodePanel";
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
      className="calc-visualizer"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="calc-viz-tabs">
        {CALC_EXAMPLES.map((ex, i) => (
          <button
            key={ex.id}
            className={`calc-viz-tab${i === activeTab ? " calc-viz-tab--active" : ""}`}
            onClick={() => handleTabClick(i)}
          >
            {ex.title}
          </button>
        ))}
      </div>
      <div className="calc-viz-content">
        <div className="calc-viz-code">
          <CalcCodePanel
            html={codeHtmls[activeTab]}
            activeLines={step.activeLines}
          />
        </div>
        <div className="calc-viz-diagram">
          <Diagram def={example.diagram} state={diagramState} />
        </div>
      </div>
      <div className="calc-viz-progress">
        {example.steps.map((_, i) => (
          <div
            key={i}
            className={`calc-viz-dot${i === stepIndex ? " calc-viz-dot--active" : ""}`}
          />
        ))}
      </div>
    </div>
  );
};
