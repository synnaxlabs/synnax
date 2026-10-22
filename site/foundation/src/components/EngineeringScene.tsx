// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/lyra/logo";
import { type ReactElement, type ReactNode, useId } from "react";

export type EngineeringMode = "reliability" | "performance" | "deployment";
type Point = readonly [number, number, number];
type Stage = 0 | 1 | 2 | 3 | 4;

const INK = "#dce6f2";
const ACCENT = "#a8c2e3";
const EDGE = "#788faa";
const WARM = "#c4ad87";

// A shallow orthographic view keeps each connection legible through state changes.
const project = ([x, y, z]: Point): readonly [number, number] => [
  56 + x + y * 0.48,
  320 + x * 0.2 - y * 0.43 - z,
];
const coordinate = (point: Point): string => project(point).join(",");
const path = (points: readonly Point[]): string =>
  points.map((point, index) => `${index ? "L" : "M"}${coordinate(point)}`).join(" ");

const Line = ({
  points,
  color = EDGE,
  width = 1,
  dashed = false,
  opacity = 1,
}: {
  points: readonly Point[];
  color?: string;
  width?: number;
  dashed?: boolean;
  opacity?: number;
}): ReactElement => (
  <path
    d={path(points)}
    fill="none"
    stroke={color}
    strokeWidth={width}
    strokeDasharray={dashed ? "3 5" : undefined}
    opacity={opacity}
    className="eng-stroke"
  />
);

const Face = ({
  points,
  fill = "#303f55",
  stroke = EDGE,
  opacity = 1,
}: {
  points: readonly Point[];
  fill?: string;
  stroke?: string;
  opacity?: number;
}): ReactElement => (
  <polygon
    points={points.map(coordinate).join(" ")}
    fill={fill}
    stroke={stroke}
    strokeWidth="0.8"
    opacity={opacity}
    className="eng-stroke"
  />
);

const Slab = ({
  x,
  y,
  w,
  d,
  z = 0,
  thickness = 8,
  fill = "#303f55",
  stroke = EDGE,
  children,
}: {
  x: number;
  y: number;
  w: number;
  d: number;
  z?: number;
  thickness?: number;
  fill?: string;
  stroke?: string;
  children?: ReactNode;
}): ReactElement => (
  <g>
    <Face
      points={[
        [x, y, z],
        [x + w, y, z],
        [x + w, y, z - thickness],
        [x, y, z - thickness],
      ]}
      fill="#202e40"
      stroke={stroke}
    />
    <Face
      points={[
        [x + w, y, z],
        [x + w, y + d, z],
        [x + w, y + d, z - thickness],
        [x + w, y, z - thickness],
      ]}
      fill="#141e2e"
      stroke={stroke}
    />
    <Face
      points={[
        [x, y, z],
        [x + w, y, z],
        [x + w, y + d, z],
        [x, y + d, z],
      ]}
      fill={fill}
      stroke={stroke}
    />
    {children}
  </g>
);

const Label = ({
  point,
  children,
  color = "#acbdd9",
  anchor = "start",
}: {
  point: Point;
  children: ReactNode;
  color?: string;
  anchor?: "start" | "middle" | "end";
}): ReactElement => {
  const [x, y] = project(point);
  return (
    <text
      className="eng-scene-label"
      x={x}
      y={y}
      fill={color}
      textAnchor={anchor}
      fontSize="13"
      letterSpacing="1"
      fontFamily="var(--foundation-mono, 'Geist Mono', monospace)"
    >
      {children}
    </text>
  );
};

const RouteNode = ({ x }: { x: number }): ReactElement => (
  <g className="eng-motion">
    <Slab x={x} y={104} w={150} d={142} z={24} thickness={15}>
      <Face
        points={[
          [x + 12, 117, 24.5],
          [x + 138, 117, 24.5],
          [x + 138, 233, 24.5],
          [x + 12, 233, 24.5],
        ]}
        fill="#202e40"
        stroke="#526788"
      />
      {[137, 159, 181, 203].map((y, i) => (
        <g key={y}>
          <Line
            points={[
              [x + 12, y, 25],
              [x + 51, y, 25],
              [x + 74, 168 + i * 6, 25],
              [x + 138, 168 + i * 6, 25],
            ]}
            color={i === 1 ? ACCENT : "#687fa2"}
            width={i === 1 ? 1.6 : 0.9}
          />
          <Face
            points={[
              [x + 9, y - 2, 25],
              [x + 15, y - 2, 25],
              [x + 15, y + 2, 25],
              [x + 9, y + 2, 25],
            ]}
            fill={ACCENT}
            stroke={ACCENT}
          />
        </g>
      ))}
      <Slab
        x={x + 62}
        y={155}
        w={30}
        d={38}
        z={31}
        thickness={5}
        fill="#435873"
        stroke="#acbdd9"
      />
    </Slab>
    <Line
      points={[
        [x + 8, 110, 9],
        [x + 54, 110, 9],
      ]}
      color={ACCENT}
      width={2}
    />
  </g>
);

const HealthNode = ({ x, label }: { x: number; label: string }): ReactElement => {
  const [cx, cy] = project([x + 56, 177, 35]);
  const logoPlacement = { width: 44, height: 44 };
  return (
    <g>
      <Slab
        x={x}
        y={122}
        w={112}
        d={110}
        z={27}
        thickness={15}
        fill="#2b3e55"
        stroke="#829bb9"
      >
        <Face
          points={[
            [x + 12, 134, 28],
            [x + 100, 134, 28],
            [x + 100, 220, 28],
            [x + 12, 220, 28],
          ]}
          fill="#192b3f"
          stroke="#516985"
        />
        {[144, 210].map((y) => (
          <Line
            key={y}
            points={[
              [x + 15, y, 29],
              [x + 96, y, 29],
            ]}
            color="#6884a2"
          />
        ))}
      </Slab>
      <g
        transform={`translate(${cx} ${cy}) matrix(1 .2 -.48 .43 0 0) translate(-22 -22)`}
      >
        <Logo.Logo {...logoPlacement} color="auto" aria-hidden="true" />
      </g>
      <Label point={[x + 57, 279, 26]} anchor="middle" color={INK}>
        {label}
      </Label>
    </g>
  );
};

const HealthFlow = ({
  points,
  active = true,
  color = ACCENT,
  backfill = false,
}: {
  points: readonly Point[];
  active?: boolean;
  color?: string;
  backfill?: boolean;
}): ReactElement => (
  <g>
    <Line
      points={points}
      color={active ? "#607d9b" : "#786c5b"}
      width={1.3}
      dashed={!active}
    />
    {active && (
      <path
        d={path(points)}
        className={backfill ? "eng-health-flow eng-health-backfill" : "eng-health-flow"}
        fill="none"
        stroke={color}
        strokeWidth={backfill ? 2.4 : 1.8}
        strokeDasharray={backfill ? "4 12" : "10 90"}
      />
    )}
  </g>
);

const DataHealthScene = ({ stage, id }: { stage: Stage; id: string }): ReactElement => {
  const silent = stage === 1;
  const delayed = stage === 2;
  const recovering = stage === 3;
  const rangeCheck = stage === 4;
  const bufferLevel = delayed ? 4 : recovering ? 2 : 0;
  const description = [
    "Equipment feeds Ignition. Foundation site and cloud nodes carry telemetry to InfluxDB, whose write acknowledgments confirm delivery. Source quality, validation, and delivery are separate signals.",
    "The Ignition connection stays online while the stream misses its one-second update expectation. The last sample remains source-reported good, but its freshness validation is stale. The destination received that last sample.",
    "Equipment and Ignition continue to deliver fresh data to Foundation. InfluxDB writes lag by 42 seconds and 18,400 samples wait in the local buffer. Source quality remains good. Delay does not establish data loss.",
    "Live telemetry continues while retained samples catch up. 4,200 samples remain buffered. A four-second interval exceeded retention and remains an explicit gap; recovered delivery does not erase it.",
    "A fresh 86.4 bar pressure reading exceeds the configured 0 to 80 bar range. Its source-reported quality remains good, its validation is out of range, and its destination write is confirmed.",
  ][stage];
  return (
    <svg
      className="eng-scene eng-health-scene"
      viewBox="50 80 950 410"
      role="img"
      aria-labelledby={`${id}-title ${id}-description`}
      data-mode="reliability"
      data-stage={stage}
    >
      <title id={`${id}-title`}>
        Telemetry health,{" "}
        {
          [
            "flowing",
            "source silent",
            "destination delayed",
            "recovering",
            "value check",
          ][stage]
        }
      </title>
      <desc id={`${id}-description`}>{description}</desc>

      <g opacity="0.6">
        <Face
          points={[
            [0, 48, -19],
            [753, 48, -19],
            [753, 276, -19],
            [0, 276, -19],
          ]}
          fill="#111c29"
          stroke="#35485f"
        />
        {[80, 140, 200, 260].map((y) => (
          <Line
            key={y}
            points={[
              [0, y, -18],
              [753, y, -18],
            ]}
            color="#34455b"
            opacity={0.5}
          />
        ))}
        {[80, 180, 280, 380, 480, 580, 680].map((x) => (
          <Line
            key={x}
            points={[
              [x, 48, -18],
              [x, 276, -18],
            ]}
            color="#34455b"
            opacity={0.5}
          />
        ))}
      </g>

      <Label point={[37, 390, 50]} anchor="middle">
        EQUIPMENT
      </Label>
      {[21, 69].map((x, index) => (
        <g key={x}>
          <Slab
            x={x}
            y={316}
            w={33}
            d={43}
            z={32 + index * 8}
            thickness={26}
            fill="#293a4e"
            stroke="#7088a4"
          />
          <Line
            points={[
              [x + 7, 323, 33 + index * 8],
              [x + 27, 323, 33 + index * 8],
            ]}
            color={ACCENT}
            width={1.5}
          />
          <HealthFlow
            points={[
              [x + 16, 316, 6],
              [x + 16, 291, 6],
              [99, 291, 6],
              [99, 232, 27],
            ]}
          />
        </g>
      ))}

      {/* A source adapter, two routing nodes, and a destination keep distinct forms. */}
      <Slab
        x={50}
        y={122}
        w={102}
        d={110}
        z={27}
        thickness={10}
        fill="#25354a"
        stroke={silent ? WARM : EDGE}
      >
        {[0, 1, 2, 3].map((row) => (
          <Line
            key={row}
            points={[
              [66, 143 + row * 21, 28],
              [silent && row === 1 ? 108 : 137, 143 + row * 21, 28],
            ]}
            color={row === 1 ? (silent ? WARM : ACCENT) : "#65809f"}
            width={row === 1 ? 2 : 1}
          />
        ))}
      </Slab>
      <Label point={[-8, 202, -16]} color={INK} anchor="end">
        IGNITION
      </Label>
      <Label point={[100, 101, 4]} anchor="middle" color={silent ? WARM : "#a6bbd4"}>
        {silent ? "ONLINE / STALE STREAM" : "SOURCE CONNECTED"}
      </Label>

      <Slab
        x={152}
        y={162}
        w={98}
        d={27}
        z={23}
        thickness={5}
        fill="#1b2b3c"
        stroke="#526881"
      />
      <HealthFlow
        points={[
          [152, 176, 25],
          [250, 176, 25],
        ]}
        active={!silent}
        color={rangeCheck ? WARM : ACCENT}
      />
      <Slab
        x={362}
        y={162}
        w={88}
        d={27}
        z={23}
        thickness={5}
        fill="#1b2b3c"
        stroke="#526881"
      />
      <HealthFlow
        points={[
          [362, 176, 25],
          [450, 176, 25],
        ]}
        active={!silent}
      />
      <Slab
        x={562}
        y={162}
        w={88}
        d={27}
        z={23}
        thickness={5}
        fill="#1b2b3c"
        stroke={delayed ? WARM : "#526881"}
      />
      <HealthFlow
        points={[
          [562, 176, 25],
          [650, 176, 25],
        ]}
        active={!delayed && !silent}
      />
      {recovering && (
        <HealthFlow
          points={[
            [362, 187, 25],
            [450, 187, 25],
          ]}
          backfill
          color={WARM}
        />
      )}
      {recovering && (
        <HealthFlow
          points={[
            [562, 187, 25],
            [650, 187, 25],
          ]}
          backfill
          color={WARM}
        />
      )}
      <HealthNode x={250} label="FOUNDATION / SITE" />
      <HealthNode x={450} label="FOUNDATION / CLOUD" />

      <Slab
        x={650}
        y={126}
        w={88}
        d={104}
        z={27}
        thickness={22}
        fill="#24364a"
        stroke={delayed ? WARM : EDGE}
      >
        {[0, 1, 2, 3, 4].map((index) => (
          <Line
            key={index}
            points={[
              [663, 141 + index * 17, 28],
              [724, 141 + index * 17, 28],
            ]}
            color={delayed ? "#8e8069" : index === 3 ? ACCENT : "#68819d"}
            width={index === 3 ? 1.8 : 0.8}
          />
        ))}
      </Slab>
      <g className="eng-health-annotation">
        <path
          d={`M 790 230 H 938 V 302 L ${coordinate([738, 230, 27])}`}
          fill="none"
          stroke={delayed ? "#8d7d64" : "#566e87"}
          strokeWidth="1"
        />
        <text className="eng-health-callout-title" x="790" y="191">
          INFLUXDB
        </text>
        <text x="790" y="215">
          {delayed
            ? "WRITE LAG / 42 S"
            : recovering
              ? "LIVE + BACKFILL"
              : "WRITES CONFIRMED"}
        </text>
      </g>
      <Line
        points={[
          [693, 126, 2],
          [693, 92, 2],
          [506, 92, 2],
          [506, 122, 12],
        ]}
        color={delayed ? WARM : "#6d87a2"}
        dashed
        opacity={0.8}
      />
      <g className="eng-health-annotation">
        <path
          d="M 632 394 L 652 386"
          fill="none"
          stroke={delayed ? "#8d7d64" : "#566e87"}
          strokeWidth="1"
        />
        <text x="623" y="401" textAnchor="end">
          ACKNOWLEDGMENT
        </text>
      </g>

      {/* Retention grows in place, so the state changes do not rearrange the topology. */}
      <Line
        points={[
          [306, 122, 12],
          [306, 66, 12],
        ]}
        color={bufferLevel ? WARM : EDGE}
      />
      <Slab
        x={257}
        y={9}
        w={101}
        d={57}
        z={4}
        thickness={7}
        fill="#1a293b"
        stroke="#58718e"
      />
      {[0, 1, 2, 3].map((level) => (
        <g
          key={level}
          className="eng-motion"
          style={{
            opacity: level < bufferLevel ? 1 : 0.2,
            transform: `translateY(${level < bufferLevel ? -level * 7 : 0}px)`,
          }}
        >
          <Slab
            x={263}
            y={15}
            w={89}
            d={45}
            z={8 + level * 2}
            thickness={3}
            fill={bufferLevel ? "#353b41" : "#233247"}
            stroke={bufferLevel ? "#ac9c80" : "#526b86"}
          />
        </g>
      ))}
      <Label
        point={[304, -22, -9]}
        anchor="middle"
        color={bufferLevel ? WARM : "#9eb3cb"}
      >
        LOCAL BUFFER
      </Label>

      {/* The inspection trace is a stream history, not a claim of a separate dashboard. */}
      <g className="eng-health-history">
        <text x="177" y="433" fill="#a9bad0" fontSize="11" letterSpacing="1">
          STREAM HISTORY
        </text>
        <line x1="335" y1="430" x2="835" y2="430" stroke="#33475f" strokeWidth="1" />
        {Array.from({ length: 29 }, (_, i) => {
          const missing = recovering && i >= 10 && i <= 12;
          const stale = silent && i > 18;
          const pending = (delayed && i > 13) || (recovering && i > 24);
          return (
            <line
              key={i}
              x1={342 + i * 17}
              x2={342 + i * 17}
              y1="421"
              y2="439"
              stroke={
                missing ? "#c4978a" : stale ? "#736c62" : pending ? WARM : "#8ca8c8"
              }
              strokeWidth={missing || stale ? 1 : 2}
              strokeDasharray={missing || stale ? "2 3" : undefined}
              opacity={stale ? 0.7 : 1}
            />
          );
        })}
        <text
          x="835"
          y="467"
          textAnchor="end"
          fill={recovering || silent || delayed || rangeCheck ? WARM : "#a9bad0"}
          fontSize="11"
          letterSpacing=".6"
        >
          {silent
            ? "NO NEW SAMPLE / 18 S"
            : delayed
              ? "RETAINED / AWAITING DELIVERY"
              : recovering
                ? "4 S GAP / OUTSIDE RETENTION"
                : rangeCheck
                  ? "86.4 BAR / EXPECTED 0–80"
                  : "SOURCE → DESTINATION / CONFIRMED"}
        </text>
      </g>
    </svg>
  );
};

export const EngineeringScene = ({
  mode,
  stage,
}: {
  mode: EngineeringMode;
  stage: Stage;
}): ReactElement => {
  const id = useId().replace(/:/g, "");
  if (mode === "reliability") return <DataHealthScene stage={stage} id={id} />;
  const performance = mode === "performance";
  const deployment = mode === "deployment";
  const batched = performance && stage >= 1;
  const configured = deployment && stage === 2;
  const peerVisible = !deployment || stage >= 1;
  const bufferLevel = 1;
  const title = `${performance ? "The data path" : "Deployment topology"}, step ${stage + 1}`;

  return (
    <svg
      className="eng-scene"
      viewBox="0 100 900 400"
      role="img"
      aria-labelledby={`${id}-title ${id}-description`}
      data-mode={mode}
      data-stage={stage}
    >
      <title id={`${id}-title`}>{title}</title>
      <desc id={`${id}-description`}>
        {performance
          ? "Incoming channels enter the local node, are arranged into frames, and travel through the transport to the receiving node and client."
          : "Foundation nodes share the same deployment shape. Peers join the topology, then matching configuration plates appear across the nodes."}
      </desc>
      {/* One finite survey surface, rather than an infinite decorative grid. */}
      <g opacity="0.48">
        <Face
          points={[
            [26, -6, -18],
            [680, -6, -18],
            [680, 288, -18],
            [26, 288, -18],
          ]}
          fill="#141e2e"
          stroke="#35485f"
        />
        {[50, 115, 180, 245].map((y) => (
          <Line
            key={y}
            points={[
              [26, y, -17],
              [680, y, -17],
            ]}
            color="#35455f"
            opacity={0.65}
          />
        ))}
        {[80, 180, 280, 380, 480, 580, 680].map((x) => (
          <Line
            key={x}
            points={[
              [x, -6, -17],
              [x, 288, -17],
            ]}
            color="#35455f"
            opacity={0.65}
          />
        ))}
      </g>
      <Line
        points={[
          [20, 288, -18],
          [20, 309, -18],
          [682, 309, -18],
        ]}
        color="#4b5d7b"
        opacity={0.6}
      />

      {/* Additional peers share the same footprint and join from the rear plane. */}
      <g
        className="eng-motion"
        style={{
          opacity: deployment && peerVisible ? 1 : 0,
          transform:
            deployment && peerVisible ? "translate(0, 0)" : "translate(0, 22px)",
        }}
      >
        <Line
          points={[
            [304, 263, 16],
            [304, 220, 16],
            [433, 220, 16],
          ]}
          color={ACCENT}
          dashed
        />
        <Slab x={267} y={253} w={72} d={58} z={25} fill="#303f55" />
        <Slab
          x={284}
          y={267}
          w={36}
          d={27}
          z={29}
          fill="#435873"
          stroke="#acbdd9"
          thickness={3}
        />
        <Label point={[303, 335, configured ? 75 : 21]} anchor="middle">
          NODE 03
        </Label>
      </g>

      {/* A transit rail connects the routing nodes. */}
      <g
        className="eng-motion"
        style={{
          transform: "translate(0, 0)",
        }}
      >
        <Slab
          x={230}
          y={162}
          w={94}
          d={36}
          z={24}
          fill="#202e40"
          stroke={EDGE}
          thickness={5}
        />
        <Line
          points={[
            [230, 176, 25],
            [324, 176, 25],
          ]}
          color={ACCENT}
          width={1.6}
        />
        <Line
          points={[
            [230, 185, 25],
            [324, 185, 25],
          ]}
          color="#657b9f"
        />
      </g>
      <g
        className="eng-motion"
        style={{
          opacity: peerVisible ? 1 : 0.22,
          transform: "translate(0, 0)",
        }}
      >
        <Slab
          x={336}
          y={162}
          w={99}
          d={36}
          z={24}
          fill="#202e40"
          stroke={EDGE}
          thickness={5}
        />
        <Line
          points={[
            [336, 176, 25],
            [435, 176, 25],
          ]}
          color={ACCENT}
          width={1.6}
        />
        <Line
          points={[
            [336, 185, 25],
            [435, 185, 25],
          ]}
          color="#657b9f"
        />
      </g>
      <g className="eng-motion" style={{ opacity: peerVisible ? 1 : 0.22 }}>
        <Slab x={324} y={162} w={12} d={36} z={24} fill="#303f55" thickness={5} />
        <Line
          points={[
            [324, 176, 25],
            [336, 176, 25],
          ]}
          color={ACCENT}
          width={1.6}
        />
      </g>
      <g className="eng-motion" style={{ opacity: 1 }}>
        <Label point={deployment ? [331, 112, 3] : [331, 250, 28]} anchor="middle">
          UPLINK
        </Label>
      </g>

      <RouteNode x={80} />
      <g className="eng-motion" style={{ opacity: peerVisible ? 1 : 0.18 }}>
        <RouteNode x={435} />
      </g>
      <Label point={[98, 266, configured ? 76 : 28]} color={INK}>
        NODE 01
      </Label>
      <Label point={[453, 266, configured ? 76 : 28]} color={INK}>
        NODE 02
      </Label>

      {/* Incoming channels are expressed as traces, without literal device icons. */}
      {[132, 155, 178, 201].map((y, i) => (
        <g key={y}>
          <Line
            points={[
              [16, y, 24],
              [80, y, 24],
            ]}
            color={performance && stage === 0 ? ACCENT : "#657b9f"}
            width={performance && stage === 0 ? 1.5 : 0.8}
          />
          <Slab
            x={22 + i * 7}
            y={y - 3}
            w={14}
            d={6}
            z={25}
            thickness={2}
            fill={performance && stage === 0 ? "#acbdd9" : "#5a7197"}
            stroke="#acbdd9"
          />
        </g>
      ))}
      <Label point={[8, 230, 29]} anchor="middle">
        I/O
      </Label>

      {/* Local durability is a distinct stack, connected to the node by a short return. */}
      <g className="eng-motion" style={{ opacity: performance ? 0.22 : 1 }}>
        <Line
          points={[
            [147, 104, 9],
            [147, 70, 9],
          ]}
          color="#8b9fbf"
        />
        <Slab x={91} y={12} w={117} d={58} z={7} fill="#202e40" thickness={8} />
        {[0, 1, 2, 3].map((index) => {
          const visible = index < bufferLevel;
          return (
            <g
              key={index}
              className="eng-motion"
              style={{
                opacity: visible ? 1 : 0.12,
                transform: `translate(0, ${visible ? -index * 7 : 5}px)`,
              }}
            >
              <Slab
                x={96}
                y={17}
                w={107}
                d={48}
                z={11 + index * 2}
                thickness={3}
                fill="#303f55"
                stroke="#8297b7"
              >
                {[0, 1, 2, 3, 4].map((mark) => (
                  <Line
                    key={mark}
                    points={[
                      [104 + mark * 18, 26, 12 + index * 2],
                      [104 + mark * 18, 54, 12 + index * 2],
                    ]}
                    color="#6f87aa"
                  />
                ))}
              </Slab>
            </g>
          );
        })}
        <Label point={[91, -42, -6]} color="#acbdd9">
          LOCAL BUFFER
        </Label>
      </g>

      {/* Performance opens one continuous data plane into framed, parallel channels. */}
      <g
        className="eng-motion"
        style={{
          opacity: performance ? 1 : 0,
          transform: performance ? "translate(0, 0)" : "translate(0, 24px)",
        }}
      >
        <Slab
          x={254}
          y={19}
          w={323}
          d={77}
          z={11}
          thickness={7}
          fill="#202e40"
          stroke="#788faa"
        />
        {[0, 1, 2, 3].map((channel) => (
          <g
            key={channel}
            className="eng-motion"
            style={{
              transform: batched
                ? "translate(0, 0)"
                : `translate(${-channel * 6}px, ${channel * 2}px)`,
            }}
          >
            <Line
              points={[
                [267, 31 + channel * 16, 12],
                [566, 31 + channel * 16, 12],
              ]}
              color="#526a8d"
            />
            {Array.from({ length: 7 }, (_, block) => (
              <Slab
                key={block}
                x={268 + block * (batched ? 36 : 39)}
                y={27 + channel * 16}
                w={batched ? 31 : 13 + ((block + channel) % 3) * 5}
                d={9}
                z={13}
                thickness={2}
                fill={stage === 2 ? "#829fca" : batched ? "#607ba4" : "#354864"}
                stroke={stage === 2 ? ACCENT : "#93a7c7"}
              />
            ))}
          </g>
        ))}
        <g className="eng-motion" style={{ opacity: batched ? 1 : 0 }}>
          {[263, 335, 407, 479, 551].map((x) => (
            <Line
              key={x}
              points={[
                [x, 23, 19],
                [x, 91, 19],
              ]}
              color="#c9d6ec"
            />
          ))}
        </g>
        <Line
          points={[
            [229, 124, 24],
            [247, 124, 24],
            [247, 76, 11],
            [254, 76, 11],
          ]}
          color={ACCENT}
        />
        <Line
          points={[
            [577, 76, 11],
            [597, 76, 11],
            [597, 132, 24],
            [584, 132, 24],
          ]}
          color={stage === 2 ? ACCENT : "#526788"}
          width={stage === 2 ? 1.5 : 0.8}
        />
        <Label point={[270, -10, 5]}>
          {stage === 0 ? "CHANNELS" : stage === 1 ? "FRAMES" : "TRANSPORT"}
        </Label>
      </g>

      {/* The client surface is intentionally distinct from a Foundation routing node. */}
      <g
        className="eng-motion"
        style={{ opacity: deployment && stage === 0 ? 0.25 : 1 }}
      >
        <Line
          points={[
            [585, 177, 25],
            [618, 177, 25],
            [618, 211, 25],
            [637, 211, 25],
          ]}
          color={ACCENT}
          width={performance && stage === 2 ? 1.8 : 1.1}
        />
        <Slab
          x={637}
          y={179}
          w={60}
          d={65}
          z={25}
          thickness={4}
          fill="#202e40"
          stroke="#93a7c7"
        />
        {[0, 1, 2, 3].map((index) => (
          <Line
            key={index}
            points={[
              [647, 191 + index * 12, 26],
              [679 - (index % 2) * 9, 191 + index * 12, 26],
            ]}
            color={index === 1 ? ACCENT : "#6a7f9f"}
            width={1.3}
          />
        ))}
        <Label point={[666, 273, 26]} anchor="middle">
          CLIENT
        </Label>
      </g>

      {/* Configuration is a thin shared layer, not another ornamental device. */}
      {[
        { x: 93, y: 117, w: 124, d: 116 },
        { x: 448, y: 117, w: 124, d: 116 },
        { x: 274, y: 260, w: 58, d: 44 },
      ].map((plate, index) => (
        <g
          key={plate.x}
          className="eng-motion"
          style={{
            opacity: configured ? 0.96 : 0,
            transform: configured ? "translate(0, -28px)" : "translate(0, 0)",
          }}
        >
          <Slab {...plate} z={34} thickness={2} fill="#364760" stroke={ACCENT}>
            {[0, 1, 2].map((line) => (
              <Line
                key={line}
                points={[
                  [plate.x + 11, plate.y + 12 + line * 10, 35],
                  [
                    plate.x + Math.min(plate.w - 10, 38 + line * 12),
                    plate.y + 12 + line * 10,
                    35,
                  ],
                ]}
                color={line === 0 ? INK : "#a1b3d0"}
              />
            ))}
          </Slab>
          {index < 2 && (
            <Line
              points={[
                [plate.x + 4, plate.y + 4, 34],
                [plate.x + 4, plate.y + 4, 0],
              ]}
              color="#acbdd9"
              dashed
            />
          )}
        </g>
      ))}
      <g className="eng-motion" style={{ opacity: configured ? 1 : 0 }}>
        <Label point={[98, 290, 90]} anchor="middle" color={ACCENT}>
          CONFIGURATION
        </Label>
      </g>

      <g opacity="0.7">
        <Line
          points={[
            [29, -33, -18],
            [68, -33, -18],
          ]}
          color="#879ab8"
        />
        <Line
          points={[
            [29, -33, -18],
            [29, -3, -18],
          ]}
          color="#879ab8"
        />
        <Line
          points={[
            [29, -33, -18],
            [29, -33, 8],
          ]}
          color="#879ab8"
        />
      </g>
    </svg>
  );
};
