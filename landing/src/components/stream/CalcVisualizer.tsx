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
  const [remainingDuration, setRemainingDuration] = useState(0);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startTimeRef = useRef(0);
  const elapsedRef = useRef(0);
  const resumeCountRef = useRef(0);

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
    if (paused) {
      elapsedRef.current += Date.now() - startTimeRef.current;
      return;
    }
    clearTimer();
    const remaining = Math.max(0, step.duration - elapsedRef.current);
    setRemainingDuration(remaining);
    if (remaining !== step.duration) resumeCountRef.current += 1;
    startTimeRef.current = Date.now();
    timeoutRef.current = setTimeout(() => {
      elapsedRef.current = 0;
      setStepIndex((prev) => (prev + 1) % example.steps.length);
    }, remaining);
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
      elapsedRef.current = 0;
      setActiveTab(index);
      setStepIndex(0);
    },
    [activeTab],
  );

  return (
    <div
      className="calc-visualizer viz-container"
      style={{ "--play-state": paused ? "paused" : "running" } as React.CSSProperties}
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
            key={
              i === stepIndex
                ? `${activeTab}-${stepIndex}-${resumeCountRef.current}`
                : i
            }
            className={`viz-dot${i === stepIndex ? " viz-dot--active" : ""}`}
            style={
              i === stepIndex
                ? ({
                    "--step-duration": `${remainingDuration}ms`,
                  } as React.CSSProperties)
                : undefined
            }
          />
        ))}
      </div>
    </div>
  );
};
