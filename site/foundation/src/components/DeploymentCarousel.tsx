// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import {
  type KeyboardEvent,
  type ReactElement,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

import { Datum } from "@/components/deployments/DeploymentGeometry";
import { loadScenes } from "@/components/deployments/loadScenes";

const DEPLOYMENTS = [
  {
    id: "industrial",
    offsetY: -30,
    label: "Process plants",
    title: "The whole plant, in the same context.",
    description:
      "Process units, utilities, and production lines feed a shared operational picture. Foundation runs alongside local controllers and carries their data into plant-wide software.",
    signals: ["Process measurements", "Equipment state", "Production systems"],
    alt: "Distillation columns, storage tanks, utility equipment, heat exchangers, and a production hall connect through five Foundation nodes and a plant data room.",
    load: loadScenes.industrial,
  },
  {
    id: "aerospace",
    offsetY: 20,
    label: "Aerospace",
    title: "From the test bench to the launch pad.",
    description:
      "Link the hardware-in-the-loop lab, propulsion test stand, and launch infrastructure. Each site keeps its local systems while Foundation connects their telemetry and supervisory commands.",
    signals: ["HITL & avionics", "Propulsion testing", "Ground systems"],
    alt: "A rocket and service tower, a propulsion test stand with propellant tanks, and a hardware-in-the-loop lab connect through four Foundation nodes.",
    load: loadScenes.aerospace,
  },
  {
    id: "quantum",
    offsetY: 0,
    label: "Quantum labs",
    title: "A connected cryogenic facility.",
    description:
      "Dilution refrigerators, measurement racks, and experiment software share one live picture. Foundation nodes bring each cryostat into the same data plane.",
    signals: ["Cryogenic telemetry", "Instrument state", "Experiment context"],
    alt: "Five dilution refrigerators and their measurement racks connect through local Foundation nodes to a shared experiment and acquisition layer.",
    load: loadScenes.quantum,
  },
  {
    id: "energy",
    offsetY: -85,
    label: "Energy storage",
    title: "Many sites. One operating picture.",
    description:
      "Bring battery systems, power conversion, and site controls into a common data plane. Foundation nodes connect local equipment to fleet-level monitoring and dispatch software.",
    signals: ["Battery telemetry", "Power conversion", "Regional dispatch"],
    alt: "Three regional battery storage sites with battery enclosures and power equipment connect through Foundation nodes to a dispatch and substation site.",
    load: loadScenes.energy,
  },
  {
    id: "marine",
    offsetY: -85,
    label: "Marine fleets",
    title: "On every vessel. Across the fleet.",
    description:
      "Sensors, propulsion, and power systems share one onboard data layer. Foundation connects patrol vessels, support ships, and uncrewed craft to shore operations.",
    signals: ["Onboard acquisition", "Vessel systems", "Shore aggregation"],
    alt: "A patrol vessel, a support ship, and four uncrewed surface craft carry local Foundation nodes. Data routes connect the fleet to shore operations for aggregation.",
    load: loadScenes.marine,
  },
] as const;

type Scene = () => ReactElement;
interface LoadedScene {
  id: string;
  Scene?: Scene;
  failed?: boolean;
}

// Imports are cached by the browser. Hover/focus warming never renders another
// scene or grows the SVG tree, and failures remain retryable on selection.
const preload = (index: number): void => {
  void DEPLOYMENTS[index].load().catch(() => undefined);
};

export const DeploymentCarousel = (): ReactElement => {
  const [active, setActive] = useState(0);
  const [loaded, setLoaded] = useState<LoadedScene>({ id: DEPLOYMENTS[0].id });
  const [visible, setVisible] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const root = useRef<HTMLDivElement>(null);
  const tabList = useRef<HTMLDivElement>(null);
  const tabs = useRef<(HTMLButtonElement | null)[]>([]);
  const id = useId();
  const current = DEPLOYMENTS[active];
  const Scene = loaded.id === current.id ? loaded.Scene : undefined;
  const failed = loaded.id === current.id && loaded.failed;

  useEffect(() => {
    let cancelled = false;
    void current.load().then(
      (Scene) => {
        if (!cancelled) setLoaded({ id: current.id, Scene });
      },
      () => {
        if (!cancelled) setLoaded({ id: current.id, failed: true });
      },
    );
    return () => {
      cancelled = true;
    };
  }, [current, attempt]);

  useEffect(() => {
    if (!visible || Scene == null) return;
    const connection = (
      navigator as Navigator & {
        connection?: { saveData?: boolean; effectiveType?: string };
      }
    ).connection;
    if (connection?.saveData || /(^|-)2g$/.test(connection?.effectiveType ?? ""))
      return;
    const warmNeighbors = (): void => {
      preload((active + 1) % DEPLOYMENTS.length);
      preload((active + DEPLOYMENTS.length - 1) % DEPLOYMENTS.length);
    };
    if (typeof window.requestIdleCallback === "function") {
      const handle = window.requestIdleCallback(warmNeighbors);
      return () => window.cancelIdleCallback(handle);
    }
    const handle = window.setTimeout(warmNeighbors, 600);
    return () => window.clearTimeout(handle);
  }, [active, visible, Scene]);
  const change = (index: number): void =>
    setActive((index + DEPLOYMENTS.length) % DEPLOYMENTS.length);

  useEffect(() => {
    const element = root.current;
    if (element == null) return;
    if (typeof IntersectionObserver === "undefined") {
      setVisible(true);
      return;
    }
    const observer = new IntersectionObserver(
      ([entry]) => {
        element.dataset.visible = String(entry.isIntersecting);
        setVisible(entry.isIntersecting);
      },
      { rootMargin: "100px" },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const list = tabList.current;
    const selected = tabs.current[active];
    if (list == null || selected == null) return;
    const revealSelection = (): void => {
      if (list.clientWidth === 0) return;
      const left = selected.offsetLeft - 12;
      const right = selected.offsetLeft + selected.offsetWidth + 12;
      if (left < list.scrollLeft) list.scrollLeft = left;
      else if (right > list.scrollLeft + list.clientWidth)
        list.scrollLeft = right - list.clientWidth;
    };
    revealSelection();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(revealSelection);
    observer.observe(list);
    return () => observer.disconnect();
  }, [active]);

  const navigateTabs = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number,
  ): void => {
    let next: number;
    switch (event.key) {
      case "ArrowRight":
        next = (index + 1) % DEPLOYMENTS.length;
        break;
      case "ArrowLeft":
        next = (index - 1 + DEPLOYMENTS.length) % DEPLOYMENTS.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = DEPLOYMENTS.length - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    change(next);
    tabs.current[next]?.focus();
  };

  return (
    <div className="deployment-carousel" ref={root} data-visible="true">
      <div
        className="deployment-tabs"
        ref={tabList}
        role="tablist"
        aria-label="Infrastructure deployments"
      >
        {DEPLOYMENTS.map((deployment, index) => (
          <button
            key={deployment.id}
            ref={(element) => {
              tabs.current[index] = element;
            }}
            id={`${id}-tab-${deployment.id}`}
            role="tab"
            type="button"
            aria-selected={active === index}
            aria-controls={`${id}-panel`}
            tabIndex={active === index ? 0 : -1}
            onClick={() => change(index)}
            onPointerEnter={() => preload(index)}
            onFocus={() => preload(index)}
            onKeyDown={(event) => navigateTabs(event, index)}
          >
            <span aria-hidden="true">0{index + 1}</span>
            {deployment.label}
          </button>
        ))}
      </div>
      <div
        id={`${id}-panel`}
        role="tabpanel"
        aria-busy={Scene == null && !failed}
        aria-labelledby={`${id}-tab-${current.id}`}
        tabIndex={0}
      >
        <div className="deployment-canvas">
          <div className="deployment-map-header" aria-hidden="true">
            <span>FIELD STUDY → 0{active + 1}</span>
            <span className="deployment-flow-legend">
              <i />
              Foundation data flow
            </span>
          </div>
          <div className="deployment-viewport" key={current.id}>
            <svg
              className="industrial-scene deployment-scene"
              viewBox="0 -80 1500 840"
              role="img"
              aria-labelledby={`${id}-scene-title ${id}-scene-description`}
            >
              <title id={`${id}-scene-title`}>{current.title}</title>
              <desc id={`${id}-scene-description`}>{current.alt}</desc>
              <g transform={`translate(0 ${current.offsetY})`}>
                <Datum />
                {Scene != null ? (
                  <Scene />
                ) : (
                  <text
                    x="750"
                    y="350"
                    textAnchor="middle"
                    fill="#7f93ab"
                    style={{ font: "14px var(--foundation-mono)" }}
                  >
                    {failed ? "Diagram could not load." : "Loading deployment…"}
                  </text>
                )}
              </g>
            </svg>
          </div>
          <div className="deployment-map-footer" aria-hidden="true">
            <span>SCHEMATIC DEPLOYMENT</span>
            <svg viewBox="0 0 80 64" className="deployment-axes">
              <path d="M36 38L65 50M36 38L12 48M36 38V8" />
              <text x="68" y="55">
                X
              </text>
              <text x="2" y="52">
                Y
              </text>
              <text x="32" y="6">
                Z
              </text>
            </svg>
          </div>
        </div>
        <div className="deployment-caption">
          <div
            className="deployment-caption-copy"
            aria-live="polite"
            aria-atomic="true"
          >
            <h3>{current.title}</h3>
            <p>{current.description}</p>
            <ul aria-label="Connected systems">
              {current.signals.map((signal) => (
                <li key={signal}>{signal}</li>
              ))}
            </ul>
          </div>
          <div className="deployment-pagination" aria-label="Browse deployments">
            {failed && (
              <button
                type="button"
                aria-label="Retry loading deployment"
                onClick={() => {
                  setLoaded({ id: current.id });
                  setAttempt((value) => value + 1);
                }}
              >
                <Icon.Refresh aria-hidden="true" />
              </button>
            )}
            <span className="deployment-count" aria-hidden="true">
              0{active + 1} <span>of</span> 0{DEPLOYMENTS.length}
            </span>
            <button
              type="button"
              aria-label="Previous deployment"
              onClick={() => change(active - 1)}
            >
              <Icon.Arrow.Left aria-hidden="true" />
            </button>
            <button
              type="button"
              aria-label="Next deployment"
              onClick={() => change(active + 1)}
            >
              <Icon.Arrow.Right aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
