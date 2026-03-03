import { type ReactElement, useEffect, useRef } from "react";

const COLS = 50;
const ROWS = 20;
const COL_SPACING = 8;
const ROW_SPACING = 7;
const STAGGER = COL_SPACING / 2;
const WAVE_WIDTH = 10;
const CONNECTION_DIST = 12;
const NODE_MIN_R = 0.5;
const NODE_MAX_R = 1.4;

interface Color {
  r: number;
  g: number;
  b: number;
}

const COLOR_AUTO: Color = { r: 55, g: 116, b: 208 };
const COLOR_ABORT: Color = { r: 220, g: 70, b: 70 };
const COLOR_MANUAL: Color = { r: 80, g: 200, b: 120 };

const LEGEND = [
  { label: "automated", color: COLOR_AUTO },
  { label: "abort", color: COLOR_ABORT },
  { label: "manual", color: COLOR_MANUAL },
];

const PHASES: Array<{ from: Color; to: Color; duration: number }> = [
  { from: COLOR_AUTO, to: COLOR_ABORT, duration: 4500 },
  { from: COLOR_ABORT, to: COLOR_MANUAL, duration: 4500 },
  { from: COLOR_MANUAL, to: COLOR_AUTO, duration: 4500 },
];

const TOTAL_CYCLE = PHASES.reduce((sum, p) => sum + p.duration, 0);

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

function lerpColorRGB(
  a: Color,
  b: Color,
  t: number,
): { r: number; g: number; b: number } {
  return {
    r: Math.round(lerp(a.r, b.r, t)),
    g: Math.round(lerp(a.g, b.g, t)),
    b: Math.round(lerp(a.b, b.b, t)),
  };
}

function lerpColor(a: Color, b: Color, t: number): string {
  const c = lerpColorRGB(a, b, t);
  return `rgb(${c.r},${c.g},${c.b})`;
}

function colorStr(c: Color, opacity = 1): string {
  return `rgba(${c.r},${c.g},${c.b},${opacity})`;
}

function sigmoid(x: number): number {
  return 1 / (1 + Math.exp(-x));
}

function colorsEqual(a: Color, b: Color): boolean {
  return a.r === b.r && a.g === b.g && a.b === b.b;
}

function legendIndex(color: Color): number {
  return LEGEND.findIndex((l) => colorsEqual(l.color, color));
}

function smoothstep(t: number): number {
  const c = Math.max(0, Math.min(1, t));
  return c * c * (3 - 2 * c);
}

function hash(x: number, y: number, seed: number): number {
  let h = seed + x * 374761393 + y * 668265263;
  h = ((h ^ (h >> 13)) * 1274126177) | 0;
  return ((h ^ (h >> 16)) >>> 0) / 4294967296;
}

function noise2D(x: number, y: number, seed = 0): number {
  const ix = Math.floor(x);
  const iy = Math.floor(y);
  const fx = x - ix;
  const fy = y - iy;
  const sx = fx * fx * (3 - 2 * fx);
  const sy = fy * fy * (3 - 2 * fy);
  const a = hash(ix, iy, seed);
  const b = hash(ix + 1, iy, seed);
  const c = hash(ix, iy + 1, seed);
  const d = hash(ix + 1, iy + 1, seed);
  return lerp(lerp(a, b, sx), lerp(c, d, sx), sy);
}

interface NodeInfo {
  cx: number;
  cy: number;
  baseRadius: number;
  baseOpacity: number;
  col: number;
  row: number;
}

interface EdgeInfo {
  from: number;
  to: number;
  baseOpacity: number;
}

function buildGraph(
  gridWidth: number,
  gridHeight: number,
): { nodes: NodeInfo[]; edges: EdgeInfo[] } {
  const centerX = gridWidth / 2;
  const centerY = gridHeight * 0.47;
  const nodes: NodeInfo[] = [];

  for (let row = 0; row < ROWS; row++) {
    for (let col = 0; col < COLS; col++) {
      const isOddRow = row % 2 === 1;
      const baseCx =
        COL_SPACING / 2 + (isOddRow ? STAGGER : 0) + col * COL_SPACING;
      const baseCy = ROW_SPACING / 2 + row * ROW_SPACING;

      const jx =
        (noise2D(col * 0.5, row * 0.5, 42) - 0.5) * COL_SPACING * 0.35;
      const jy =
        (noise2D(col * 0.5, row * 0.5, 137) - 0.5) * ROW_SPACING * 0.35;
      const cx = baseCx + jx;
      const cy = baseCy + jy;

      const dx = (cx - centerX) / (gridWidth * 0.52);
      const dy = (cy - centerY) / (gridHeight * 0.48);
      const dist = Math.sqrt(dx * dx + dy * dy);

      const edgeNoise = noise2D(col * 0.15, row * 0.15, 77) * 0.25 - 0.12;
      const adjustedDist = dist + edgeNoise;

      const plateauEnd = 0.2;
      const fadeEnd = 0.92;
      let vignette: number;
      if (adjustedDist <= plateauEnd) vignette = 1.0;
      else if (adjustedDist >= fadeEnd) vignette = 0.0;
      else {
        const ft = (adjustedDist - plateauEnd) / (fadeEnd - plateauEnd);
        vignette = 1.0 - smoothstep(ft);
      }

      const sizeNoise =
        1.0 + (noise2D(col * 0.2, row * 0.2, 99) - 0.5) * 0.8;
      const baseRadius = lerp(NODE_MIN_R, NODE_MAX_R, vignette) * sizeNoise;
      const baseOpacity = lerp(0.02, 0.2, vignette);

      if (vignette > 0.03) {
        nodes.push({ cx, cy, baseRadius, baseOpacity, col, row });
      }
    }
  }

  const edges: EdgeInfo[] = [];
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const a = nodes[i];
      const b = nodes[j];
      const ddx = a.cx - b.cx;
      const ddy = a.cy - b.cy;
      const d = Math.sqrt(ddx * ddx + ddy * ddy);
      if (d < CONNECTION_DIST) {
        const proximity = 1.0 - d / CONNECTION_DIST;
        const avgOpacity = (a.baseOpacity + b.baseOpacity) / 2;
        edges.push({
          from: i,
          to: j,
          baseOpacity: proximity * avgOpacity * 0.6,
        });
      }
    }
  }

  return { nodes, edges };
}

export function ControlHandoffGraphic({
  className,
}: {
  className?: string;
}): ReactElement {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const barRef = useRef<HTMLDivElement>(null);
  const highlightRef = useRef<HTMLDivElement>(null);
  const legendTextsRef = useRef<HTMLSpanElement[]>([]);
  const animRef = useRef<number>(0);
  const visibleRef = useRef(false);

  const gridWidth = COLS * COL_SPACING + STAGGER;
  const gridHeight = ROWS * ROW_SPACING;
  const w = gridWidth;

  const graphData = useRef(buildGraph(gridWidth, gridHeight)).current;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas == null) return;
    const ctx = canvas.getContext("2d");
    if (ctx == null) return;

    const dpr = window.devicePixelRatio || 1;
    const resizeCanvas = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = rect.width * dpr;
      canvas.height = rect.height * dpr;
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.scale(
        (rect.width * dpr) / w,
        (rect.height * dpr) / gridHeight,
      );
    };
    resizeCanvas();

    const resizeObserver = new ResizeObserver(resizeCanvas);
    resizeObserver.observe(canvas);

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          visibleRef.current = entry.isIntersecting;
          if (entry.isIntersecting && animRef.current === 0) startAnimation();
        });
      },
      { threshold: 0.3 },
    );
    observer.observe(canvas);

    function startAnimation() {
      let start: number | null = null;

      function tick(timestamp: number) {
        if (!visibleRef.current) {
          animRef.current = 0;
          return;
        }
        if (start == null) start = timestamp;
        if (ctx == null) return;
        const elapsed = timestamp - start;

        let cycleTime = elapsed % TOTAL_CYCLE;
        let phaseIndex = 0;
        let phaseProgress = 0;
        for (let i = 0; i < PHASES.length; i++) {
          if (cycleTime < PHASES[i].duration) {
            phaseIndex = i;
            phaseProgress = cycleTime / PHASES[i].duration;
            break;
          }
          cycleTime -= PHASES[i].duration;
        }

        const phase = PHASES[phaseIndex];
        const waveFront =
          phaseProgress * (COLS + WAVE_WIDTH * 2) - WAVE_WIDTH;

        const nodeR: number[] = [];
        const nodeG: number[] = [];
        const nodeB: number[] = [];
        const nodeFlash: number[] = [];
        const nodeRadius: number[] = [];
        const nodeOpacity: number[] = [];

        for (let i = 0; i < graphData.nodes.length; i++) {
          const info = graphData.nodes[i];
          const rowWobble =
            Math.sin(info.row * 0.5 + info.col * 0.1) * 1.5;
          const dist = info.col - waveFront + rowWobble;
          const t = sigmoid(-dist / (WAVE_WIDTH * 0.3));
          const c = lerpColorRGB(phase.from, phase.to, t);
          nodeR[i] = c.r;
          nodeG[i] = c.g;
          nodeB[i] = c.b;

          const atFront = Math.abs(dist) < WAVE_WIDTH * 0.7;
          const flash = atFront
            ? Math.exp(-((dist * dist) / (WAVE_WIDTH * 1.2)))
            : 0;
          nodeFlash[i] = flash;
          nodeRadius[i] = lerp(info.baseRadius, info.baseRadius * 2.5, flash);
          nodeOpacity[i] = lerp(info.baseOpacity, 0.85, flash);
        }

        ctx.clearRect(0, 0, w, gridHeight);

        for (let i = 0; i < graphData.edges.length; i++) {
          const edge = graphData.edges[i];
          const avgFlash =
            (nodeFlash[edge.from] + nodeFlash[edge.to]) / 2;
          const opacity = lerp(edge.baseOpacity, 0.5, avgFlash);
          if (opacity < 0.005) continue;

          const a = graphData.nodes[edge.from];
          const b = graphData.nodes[edge.to];
          ctx.strokeStyle = `rgba(${nodeR[edge.from]},${nodeG[edge.from]},${nodeB[edge.from]},${opacity})`;
          ctx.lineWidth = 0.35;
          ctx.beginPath();
          ctx.moveTo(a.cx, a.cy);
          ctx.lineTo(b.cx, b.cy);
          ctx.stroke();
        }

        for (let i = 0; i < graphData.nodes.length; i++) {
          const info = graphData.nodes[i];
          const opacity = nodeOpacity[i];
          if (opacity < 0.005) continue;

          ctx.fillStyle = `rgba(${nodeR[i]},${nodeG[i]},${nodeB[i]},${opacity})`;
          ctx.beginPath();
          ctx.arc(info.cx, info.cy, nodeRadius[i], 0, Math.PI * 2);
          ctx.fill();
        }

        const fromIdx = legendIndex(phase.from);
        const toIdx = legendIndex(phase.to);
        const barWidth = barRef.current?.offsetWidth ?? 1;
        const secWidth = barWidth / LEGEND.length;
        const fromCenter = fromIdx * secWidth + secWidth / 2;
        const toCenter = toIdx * secWidth + secWidth / 2;
        const highlightPx =
          lerp(fromCenter, toCenter, phaseProgress);

        if (highlightRef.current != null) {
          const currentColor = lerpColor(phase.from, phase.to, phaseProgress);
          highlightRef.current.style.left = `${highlightPx}px`;
          highlightRef.current.style.background =
            `radial-gradient(ellipse at center, ${currentColor.replace("rgb", "rgba").replace(")", ",0.3)")} 0%, transparent 70%)`;
        }

        const midpoint = 0.5;
        const activeIdx = phaseProgress < midpoint ? fromIdx : toIdx;
        for (let i = 0; i < LEGEND.length; i++) {
          const legendText = legendTextsRef.current[i];
          if (legendText != null) {
            legendText.style.color =
              i === activeIdx
                ? "var(--pluto-gray-l9)"
                : "var(--pluto-gray-l4)";
          }
        }

        animRef.current = requestAnimationFrame(tick);
      }

      animRef.current = requestAnimationFrame(tick);
    }

    return () => {
      resizeObserver.disconnect();
      observer.disconnect();
      if (animRef.current !== 0) cancelAnimationFrame(animRef.current);
    };
  }, []);

  const barGradient = LEGEND.map(
    (item, i) =>
      `${colorStr(item.color, 0.12)} ${(i / (LEGEND.length - 1)) * 100}%`,
  ).join(", ");

  return (
    <div className={className}>
      <canvas
        ref={canvasRef}
        style={{
          width: "100%",
          aspectRatio: `${w} / ${gridHeight}`,
          display: "block",
        }}
      />
      <div
        ref={barRef}
        className="handoff-bar"
        style={{ background: `linear-gradient(to right, ${barGradient})` }}
      >
        <div ref={highlightRef} className="handoff-bar-highlight" />
        {LEGEND.map((item, i) => (
          <span
            key={item.label}
            ref={(el) => {
              if (el != null) legendTextsRef.current[i] = el;
            }}
            className="handoff-bar-label"
          >
            {item.label}
          </span>
        ))}
      </div>
    </div>
  );
}
