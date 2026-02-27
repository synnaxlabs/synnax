import { useEffect, useRef } from "react";

export function ControlHandoffGraphic({ className }: { className?: string }) {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const svg = svgRef.current;
    if (!svg) return;
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) svg.classList.add("animated");
        });
      },
      { threshold: 0.5 },
    );
    observer.observe(svg);
    return () => observer.disconnect();
  }, []);

  return (
    <div className={className}>
      <div className="handoff-entities">
        <Entity name="Automated" />
        <Entity name="Abort" />
        <Entity name="Manual" />
      </div>
      <svg
        ref={svgRef}
        className="handoff-paths"
        xmlns="http://www.w3.org/2000/svg"
        height="178"
        viewBox="0 0 483 176"
        fill="none"
        preserveAspectRatio="none"
      >
        <path
          className="handoff-path-1"
          vectorEffect="non-scaling-stroke"
          d="M0 1.00005C0 1.00005 26.775 0.999936 116.55 1.00005C206.325 1.00017 180.6 175 240.45 175C300.3 175 254.413 175 300.825 175C351.75 175 332.325 88.2493 379.05 88.2493C425.775 88.2493 483 88.2493 483 88.2493"
          stroke="#3774D0"
          strokeWidth={2}
        />
        <path
          className="handoff-path-2"
          vectorEffect="non-scaling-stroke"
          d="M1 175C1 175 5.22999 175 35.4643 175C79.9183 175 81.4167 1 116.381 1C140.763 1 259.824 2.4957 284.207 2.4957C324.165 2.4957 322.418 88.2494 361.627 88.2493C408.079 88.2492 458.526 88.2492 484 88.2493"
          stroke="#50C878"
          strokeWidth={2}
          strokeLinecap="round"
        />
      </svg>
    </div>
  );
}

function Entity({ name }: { name: string }) {
  return (
    <div className="handoff-entity">
      <div className="handoff-chip">
        <span>{name}</span>
      </div>
      <div className="handoff-line" />
    </div>
  );
}
