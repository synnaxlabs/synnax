export const manhattanPath = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  r: number = 8,
): string => {
  const midX = (x1 + x2) / 2;
  if (Math.abs(y1 - y2) < 1) return `M${x1},${y1} L${x2},${y2}`;
  const dy = y2 > y1 ? 1 : -1;
  const absD = Math.abs(y2 - y1);
  const cr = Math.min(r, absD / 2, Math.abs(midX - x1) / 2);
  return [
    `M${x1},${y1}`,
    `L${midX - cr},${y1}`,
    `Q${midX},${y1} ${midX},${y1 + dy * cr}`,
    `L${midX},${y2 - dy * cr}`,
    `Q${midX},${y2} ${midX + cr},${y2}`,
    `L${x2},${y2}`,
  ].join(" ");
};

export const manhattanMidpoint = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): { x: number; y: number } => ({
  x: (x1 + x2) / 2,
  y: (y1 + y2) / 2,
});
