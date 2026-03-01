import { type ReactElement, useEffect, useRef } from "react";

const permutation = new Uint8Array(512);
const grad = [
  [1, 1], [-1, 1], [1, -1], [-1, -1],
  [1, 0], [-1, 0], [0, 1], [0, -1],
];

const initNoise = (seed: number): void => {
  const p = new Uint8Array(256);
  for (let i = 0; i < 256; i++) p[i] = i;
  let s = seed;
  for (let i = 255; i > 0; i--) {
    s = (s * 16807 + 0) % 2147483647;
    const j = s % (i + 1);
    [p[i], p[j]] = [p[j], p[i]];
  }
  for (let i = 0; i < 512; i++) permutation[i] = p[i & 255];
};

const fade = (t: number): number => t * t * t * (t * (t * 6 - 15) + 10);
const lerp = (a: number, b: number, t: number): number => a + t * (b - a);
const dot2 = (g: number[], x: number, y: number): number => g[0] * x + g[1] * y;

const perlin2 = (x: number, y: number): number => {
  const xi = Math.floor(x) & 255;
  const yi = Math.floor(y) & 255;
  const xf = x - Math.floor(x);
  const yf = y - Math.floor(y);
  const u = fade(xf);
  const v = fade(yf);
  const aa = permutation[permutation[xi] + yi] & 7;
  const ab = permutation[permutation[xi] + yi + 1] & 7;
  const ba = permutation[permutation[xi + 1] + yi] & 7;
  const bb = permutation[permutation[xi + 1] + yi + 1] & 7;
  return lerp(
    lerp(dot2(grad[aa], xf, yf), dot2(grad[ba], xf - 1, yf), u),
    lerp(dot2(grad[ab], xf, yf - 1), dot2(grad[bb], xf - 1, yf - 1), u),
    v,
  );
};

const fbm = (x: number, y: number, octaves: number): number => {
  let val = 0;
  let amp = 1;
  let freq = 1;
  let max = 0;
  for (let i = 0; i < octaves; i++) {
    val += perlin2(x * freq, y * freq) * amp;
    max += amp;
    amp *= 0.5;
    freq *= 2;
  }
  return val / max;
};

type Point = [number, number];

const marchSegments = (
  field: Float64Array,
  w: number,
  h: number,
  threshold: number,
): Array<[Point, Point]> => {
  const segments: Array<[Point, Point]> = [];
  for (let y = 0; y < h - 1; y++) {
    for (let x = 0; x < w - 1; x++) {
      const tl = field[y * w + x];
      const tr = field[y * w + x + 1];
      const br = field[(y + 1) * w + x + 1];
      const bl = field[(y + 1) * w + x];

      let c = 0;
      if (tl > threshold) c |= 8;
      if (tr > threshold) c |= 4;
      if (br > threshold) c |= 2;
      if (bl > threshold) c |= 1;
      if (c === 0 || c === 15) continue;

      const t = (a: number, b: number): number => {
        const d = b - a;
        return d === 0 ? 0.5 : (threshold - a) / d;
      };

      const top: Point = [x + t(tl, tr), y];
      const right: Point = [x + 1, y + t(tr, br)];
      const bottom: Point = [x + t(bl, br), y + 1];
      const left: Point = [x, y + t(tl, bl)];

      switch (c) {
        case 1: segments.push([left, bottom]); break;
        case 2: segments.push([bottom, right]); break;
        case 3: segments.push([left, right]); break;
        case 4: segments.push([top, right]); break;
        case 5: segments.push([left, top]); segments.push([bottom, right]); break;
        case 6: segments.push([top, bottom]); break;
        case 7: segments.push([left, top]); break;
        case 8: segments.push([top, left]); break;
        case 9: segments.push([top, bottom]); break;
        case 10: segments.push([top, right]); segments.push([left, bottom]); break;
        case 11: segments.push([top, right]); break;
        case 12: segments.push([left, right]); break;
        case 13: segments.push([bottom, right]); break;
        case 14: segments.push([left, bottom]); break;
      }
    }
  }
  return segments;
};

const chainSegments = (segments: Array<[Point, Point]>): Point[][] => {
  const EPS = 1e-6;
  const key = (p: Point): string => `${p[0].toFixed(4)},${p[1].toFixed(4)}`;

  // Build adjacency: each endpoint maps to list of segment indices
  const adj = new Map<string, number[]>();
  for (let i = 0; i < segments.length; i++) {
    for (const pt of segments[i]) {
      const k = key(pt);
      const list = adj.get(k);
      if (list != null) list.push(i);
      else adj.set(k, [i]);
    }
  }

  const used = new Uint8Array(segments.length);
  const paths: Point[][] = [];

  for (let i = 0; i < segments.length; i++) {
    if (used[i]) continue;
    used[i] = 1;
    const path: Point[] = [segments[i][0], segments[i][1]];

    // Extend forward
    let extended = true;
    while (extended) {
      extended = false;
      const tail = path[path.length - 1];
      const tk = key(tail);
      const neighbors = adj.get(tk);
      if (neighbors == null) break;
      for (const ni of neighbors) {
        if (used[ni]) continue;
        const seg = segments[ni];
        const d0 = Math.abs(seg[0][0] - tail[0]) + Math.abs(seg[0][1] - tail[1]);
        const d1 = Math.abs(seg[1][0] - tail[0]) + Math.abs(seg[1][1] - tail[1]);
        if (d0 < EPS) {
          path.push(seg[1]);
          used[ni] = 1;
          extended = true;
          break;
        } else if (d1 < EPS) {
          path.push(seg[0]);
          used[ni] = 1;
          extended = true;
          break;
        }
      }
    }

    // Extend backward
    extended = true;
    while (extended) {
      extended = false;
      const head = path[0];
      const hk = key(head);
      const neighbors = adj.get(hk);
      if (neighbors == null) break;
      for (const ni of neighbors) {
        if (used[ni]) continue;
        const seg = segments[ni];
        const d0 = Math.abs(seg[0][0] - head[0]) + Math.abs(seg[0][1] - head[1]);
        const d1 = Math.abs(seg[1][0] - head[0]) + Math.abs(seg[1][1] - head[1]);
        if (d0 < EPS) {
          path.unshift(seg[1]);
          used[ni] = 1;
          extended = true;
          break;
        } else if (d1 < EPS) {
          path.unshift(seg[0]);
          used[ni] = 1;
          extended = true;
          break;
        }
      }
    }

    if (path.length >= 3) paths.push(path);
  }

  return paths;
};

const CONTOUR_LEVELS = 9;
const SCALE = 0.004;
const OCTAVES = 3;
const SEED = 5;
const COLOR_SEED = 256;
const GRAY = "rgba(36, 36, 41, 0.5)";
const BLUE = "rgba(59, 130, 246, 0.3)";
const GRID_STEP = 3;

export const TopoCanvas = (): ReactElement => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas == null) return;
    const ctx = canvas.getContext("2d");
    if (ctx == null) return;

    const draw = (): void => {
      const { offsetWidth: cw, offsetHeight: ch } = canvas;
      const dpr = window.devicePixelRatio || 1;
      canvas.width = cw * dpr;
      canvas.height = ch * dpr;
      ctx.scale(dpr, dpr);
      ctx.clearRect(0, 0, cw, ch);

      const gw = Math.ceil(cw / GRID_STEP) + 1;
      const gh = Math.ceil(ch / GRID_STEP) + 1;
      const field = new Float64Array(gw * gh);

      initNoise(SEED);
      for (let gy = 0; gy < gh; gy++)
        for (let gx = 0; gx < gw; gx++)
          field[gy * gw + gx] = fbm(gx * GRID_STEP * SCALE, gy * GRID_STEP * SCALE, OCTAVES);

      // Color noise field
      initNoise(COLOR_SEED);
      const colorField = new Float64Array(gw * gh);
      for (let gy = 0; gy < gh; gy++)
        for (let gx = 0; gx < gw; gx++)
          colorField[gy * gw + gx] = fbm(gx * GRID_STEP * 0.003, gy * GRID_STEP * 0.003, 2);

      let min = Infinity;
      let max = -Infinity;
      for (let i = 0; i < field.length; i++) {
        if (field[i] < min) min = field[i];
        if (field[i] > max) max = field[i];
      }

      ctx.lineWidth = 1;
      ctx.lineJoin = "round";
      ctx.lineCap = "round";

      for (let level = 1; level < CONTOUR_LEVELS; level++) {
        const threshold = min + (level / CONTOUR_LEVELS) * (max - min);
        const segments = marchSegments(field, gw, gh, threshold);
        const paths = chainSegments(segments);

        for (const path of paths) {
          // Determine color from midpoint of path
          const mid = path[Math.floor(path.length / 2)];
          const mx = Math.floor(mid[0]);
          const my = Math.floor(mid[1]);
          const ci = my * gw + mx;
          const isBlue = ci >= 0 && ci < colorField.length && colorField[ci] > 0.15;

          ctx.strokeStyle = isBlue ? BLUE : GRAY;
          ctx.beginPath();
          ctx.moveTo(path[0][0] * GRID_STEP, path[0][1] * GRID_STEP);
          for (let i = 1; i < path.length; i++)
            ctx.lineTo(path[i][0] * GRID_STEP, path[i][1] * GRID_STEP);
          ctx.stroke();
        }
      }
    };

    draw();
    const onResize = (): void => draw();
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  return <canvas ref={canvasRef} className="contact-topo" />;
};
