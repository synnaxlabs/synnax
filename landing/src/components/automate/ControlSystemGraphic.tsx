export function ControlSystemGraphic({
  className,
  size = 400,
}: {
  className?: string;
  size?: number;
}) {
  const cx = size / 2;
  const cy = size / 2;
  const r = size * 0.38;

  function spiralPath(
    startAngle: number,
    turns: number,
    startRadius: number,
    direction: 1 | -1 = 1,
  ) {
    const points: [number, number][] = [];
    const steps = 120;
    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      const decay = Math.exp(-3.2 * t);
      const currentRadius = startRadius * decay;
      const angle = startAngle + direction * turns * 2 * Math.PI * t;
      const x = cx + currentRadius * Math.cos(angle);
      const y = cy + currentRadius * Math.sin(angle);
      points.push([x, y]);
    }
    return points;
  }

  function pointsToSmoothPath(points: [number, number][]) {
    if (points.length < 2) return "";
    let d = `M ${points[0][0]} ${points[0][1]}`;
    for (let i = 1; i < points.length - 1; i++) {
      const xc = (points[i][0] + points[i + 1][0]) / 2;
      const yc = (points[i][1] + points[i + 1][1]) / 2;
      d += ` Q ${points[i][0]} ${points[i][1]} ${xc} ${yc}`;
    }
    const last = points[points.length - 1];
    d += ` L ${last[0]} ${last[1]}`;
    return d;
  }

  const trajectories = [
    { startAngle: 0, turns: 1.8, radius: r, direction: 1 as const, color: "#c4c4c4" },
    {
      startAngle: Math.PI * 0.4,
      turns: 2.1,
      radius: r * 0.95,
      direction: -1 as const,
      color: "#9a9a9a",
    },
    {
      startAngle: Math.PI * 0.8,
      turns: 1.5,
      radius: r * 0.88,
      direction: 1 as const,
      color: "#b0b0b0",
    },
    {
      startAngle: Math.PI * 1.2,
      turns: 2.4,
      radius: r * 0.92,
      direction: -1 as const,
      color: "#8a8a8a",
    },
    {
      startAngle: Math.PI * 1.6,
      turns: 1.9,
      radius: r * 0.97,
      direction: 1 as const,
      color: "#a8a8a8",
    },
  ];

  const paths = trajectories.map((t) => ({
    d: pointsToSmoothPath(spiralPath(t.startAngle, t.turns, t.radius, t.direction)),
    color: t.color,
  }));

  const dotRadius = size * 0.012;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      role="img"
      aria-label="Converging trajectories representing a real-time control system"
    >
      {[0.25, 0.5, 0.75].map((scale) => (
        <circle
          key={scale}
          cx={cx}
          cy={cy}
          r={r * scale}
          stroke="#e0e0e0"
          strokeWidth={0.5}
          strokeDasharray="2 4"
          opacity={0.4}
        />
      ))}
      {paths.map((path, i) => (
        <path
          key={i}
          d={path.d}
          stroke={path.color}
          strokeWidth={1.5}
          strokeLinecap="round"
          opacity={0.7}
        />
      ))}
      {trajectories.map((t, i) => {
        const pts = spiralPath(t.startAngle, t.turns, t.radius, t.direction);
        const idx = Math.floor(pts.length * 0.82);
        const idxNext = Math.min(idx + 3, pts.length - 1);
        const [ax, ay] = pts[idx];
        const [bx, by] = pts[idxNext];
        const angle = Math.atan2(by - ay, bx - ax);
        const arrowLen = size * 0.018;
        const arrowSpread = 0.5;
        const x1 = ax - arrowLen * Math.cos(angle - arrowSpread);
        const y1 = ay - arrowLen * Math.sin(angle - arrowSpread);
        const x2 = ax - arrowLen * Math.cos(angle + arrowSpread);
        const y2 = ay - arrowLen * Math.sin(angle + arrowSpread);
        return (
          <polygon
            key={`arrow-${i}`}
            points={`${ax},${ay} ${x1},${y1} ${x2},${y2}`}
            fill={paths[i].color}
            opacity={0.7}
          />
        );
      })}
      <circle cx={cx} cy={cy} r={dotRadius} fill="#555555" />
      <circle
        cx={cx}
        cy={cy}
        r={dotRadius * 2.5}
        stroke="#aaaaaa"
        strokeWidth={0.8}
        opacity={0.3}
      />
    </svg>
  );
}
