import { CodePanel } from "@/components/automate/CodePanel";
import { AbortDiagram } from "@/components/automate/diagrams/AbortDiagram";
import { AlarmDiagram } from "@/components/automate/diagrams/AlarmDiagram";
import { PressureDiagram } from "@/components/automate/diagrams/PressureDiagram";
import {
  type DiagramState,
  EXAMPLES,
  ZERO_DIAGRAM_STATE,
} from "@/components/automate/timeline";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

interface AutomateVisualizerProps {
  codeHtmls: string[];
}

const DIAGRAMS: Record<string, React.FC<{ state: DiagramState }>> = {
  pressure: PressureDiagram,
  alarm: AlarmDiagram,
  abort: AbortDiagram,
};

export const AutomateVisualizer = ({
  codeHtmls,
}: AutomateVisualizerProps): ReactElement => {
  const [activeTab, setActiveTab] = useState(0);
  const [stepIndex, setStepIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const example = EXAMPLES[activeTab];
  const step = example.steps[stepIndex];
  const diagramState: DiagramState = { ...ZERO_DIAGRAM_STATE, ...step.state };
  const DiagramComponent = DIAGRAMS[example.id];

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
  }, [stepIndex, activeTab, paused, example.steps.length, step.duration, clearTimer]);

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
      className="automate-visualizer"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="automate-viz-tabs">
        {EXAMPLES.map((ex, i) => (
          <button
            key={ex.id}
            className={`automate-viz-tab${i === activeTab ? " automate-viz-tab--active" : ""}`}
            onClick={() => handleTabClick(i)}
          >
            {ex.title}
          </button>
        ))}
      </div>
      <div className="automate-viz-content">
        <div className="automate-viz-code">
          <CodePanel html={codeHtmls[activeTab]} activeLines={step.activeLines} />
        </div>
        <div className="automate-viz-diagram">
          {DiagramComponent != null && <DiagramComponent state={diagramState} />}
        </div>
      </div>
      <div className="automate-viz-progress">
        {example.steps.map((_, i) => (
          <div
            key={i}
            className={`automate-viz-dot${i === stepIndex ? " automate-viz-dot--active" : ""}`}
          />
        ))}
      </div>
    </div>
  );
};
