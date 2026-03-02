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
  const [tick, setTick] = useState(0);
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
  }, [stepIndex, activeTab, paused, tick, example.steps.length, step.duration, clearTimer]);

  const handleTabClick = useCallback(
    (index: number) => {
      if (index === activeTab) return;
      setPaused(false);
      setActiveTab(index);
      setStepIndex(0);
    },
    [activeTab],
  );

  const handleMouseEnter = useCallback(() => setPaused(true), []);
  const handleMouseLeave = useCallback(() => {
    setPaused(false);
    setTick((t) => t + 1);
  }, []);

  return (
    <div
      className="calc-visualizer viz-container"
      style={{ "--play-state": paused ? "paused" : "running" } as React.CSSProperties}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
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
        {example.steps.map((s, i) => (
          <div
            key={i === stepIndex ? `${activeTab}-${stepIndex}-${tick}` : i}
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
