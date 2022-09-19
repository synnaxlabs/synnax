import { motion } from "framer-motion";
import { useRef } from "react";

type Range = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10;

export default function PhaserPath({
  start = [0, 0],
  points,
  count = 1,
  spacing = 5,
  strokeWidth = 2,
  phaserStrokeColor = "#3874D1",
  phaserLength = 20,
  backgroundColor = "rgba(0,0,0,0.1)",
  speed = 5,
  endLine = true,
  startLine = true,
  startEndLineExtra = 10,
  repeatType = "loop",
}: {
  spacing?: number;
  count?: number;
  startLine?: boolean;
  endLine?: boolean;
  backgroundColor?: string;
  startEndLineExtra?: number;
  phaserLength?: number;
  strokeWidth?: number;
  speed?: Range;
  phaserStrokeColor?: string;
  repeatType?: "loop" | "reverse";
  start: [number, number];
  points: [number, number, boolean][];
}) {
  const path = useRef<SVGPathElement | null>(null);

  const pathStrings = buildPathString({ spacing, count, start, points });

  const animate = {
    initial: { strokeDashoffset: 0 },
    phaser: (i: number) => {
      return {
        strokeDashoffset:
          (path.current?.getTotalLength() || 0) + phaserLength / 2,
        transition: {
          duration: speed * 3,
          ease: "linear",
          repeat: Infinity,
          repeatType: repeatType,
        },
      };
    },
  };

  return (
    <>
      {startLine && (
        <motion.line
          y1={start[1] + count * spacing + startEndLineExtra}
          y2={start[1] - spacing - startEndLineExtra}
          x1={start[0]}
          x2={start[0]}
          strokeWidth={strokeWidth + 1}
          stroke={backgroundColor}
        />
      )}
      {pathStrings.map((pathString, i) => (
        <>
          <motion.path
            ref={path}
            d={pathString}
            fill="none"
            stroke={backgroundColor}
            strokeWidth={strokeWidth}
          />
          <motion.path
            ref={path}
            d={pathString}
            custom={i}
            variants={animate}
            animate="phaser"
            initial="initial"
            fill="none"
            stroke={phaserStrokeColor}
            strokeDasharray={phaserLength}
            strokeWidth={strokeWidth}
          />
        </>
      ))}
      {endLine && (
        <motion.line
          y1={
            points[points.length - 1][1] + count * spacing + startEndLineExtra
          }
          y2={points[points.length - 1][1] - spacing - startEndLineExtra}
          x1={points[points.length - 1][0]}
          x2={points[points.length - 1][0]}
          strokeWidth={strokeWidth + 1}
          stroke={backgroundColor}
        />
      )}
    </>
  );
}

const buildPathString = ({
  start,
  points,
  count,
  spacing,
}: {
  count: number;
  spacing: number;
  start: [number, number];
  points: [number, number, boolean][];
}) => {
  const pathStrings = Array.from({ length: count }, (_, k) => k * spacing).map(
    (offset) => {
      var path = `M ${start[0] + offset} ${start[1] + offset}`;
      if (points.length > 0) {
        if (!points[0][2]) {
          path = `M ${start[0]} ${start[1] + offset}`;
        } else {
          path = `M ${start[0] + offset} ${start[1]}`;
        }
      }
      const last = points.length - 1;
      points.forEach(([x, y, vertical], i) => {
        var horizontal = ` H ${x - offset}`;
        var vert = ` V ${y + offset}`;
        if (i === last) {
          if (!vertical) {
            vert = ` V ${y}`;
          } else {
            horizontal = ` H ${x}`;
          }
        }
        if (vertical) {
          path += vert;
          path += horizontal;
        } else {
          path += horizontal;
          path += vert;
        }
      });
      return path;
    }
  );
  return pathStrings;
};

export const HorizontalPhaser = ({
  color,
  top,
  width,
  left,
}: {
  color: string;
  top: number;
  width: number;
  left: number;
}) => {
  return (
    <PhaserPath
      startLine={false}
      endLine={false}
      start={[left + width, top]}
      points={[
        [left, top, false],
        [left, top, true],
      ]}
      strokeWidth={15}
      speed={2}
      phaserStrokeColor={color}
      count={1}
    />
  );
};
