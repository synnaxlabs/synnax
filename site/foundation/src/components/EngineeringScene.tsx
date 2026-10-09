// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useId } from "react";

export type EngineeringMode = "reliability" | "performance" | "deployment";
type Point = readonly [number, number, number];
type Stage = 0 | 1 | 2;

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

const RouteNode = ({
  x,
  muted = false,
}: {
  x: number;
  muted?: boolean;
}): ReactElement => (
  <g className="eng-motion" style={{ opacity: muted ? 0.32 : 1 }}>
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

export const EngineeringScene = ({
  mode,
  stage,
}: {
  mode: EngineeringMode;
  stage: Stage;
}): ReactElement => {
  const id = useId().replace(/:/g, "");
  const reliability = mode === "reliability";
  const performance = mode === "performance";
  const deployment = mode === "deployment";
  const disconnected = reliability && stage === 1;
  const catchingUp = reliability && stage === 2;
  const batched = performance && stage >= 1;
  const configured = deployment && stage === 2;
  const peerVisible = !deployment || stage >= 1;
  const bufferLevel = disconnected ? 4 : catchingUp ? 2 : 1;
  const title = `${mode === "reliability" ? "Distributed reliability" : performance ? "The data path" : "Deployment topology"}, step ${stage + 1}`;

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
        {reliability
          ? disconnected
            ? "The uplink is interrupted. The local node continues to collect data into its buffer while the remote node is disconnected."
            : catchingUp
              ? "The connection is restored. Buffered data moves from the local node across the uplink to the remote node and client."
              : "Two Foundation nodes exchange data across an uplink. A local buffer sits beside the source node."
          : performance
            ? "Incoming channels enter the local node, are arranged into frames, and travel through the transport to the receiving node and client."
            : "Foundation nodes share the same deployment shape. Peers join the topology, then matching configuration plates appear across the nodes."}
      </desc>
      <defs>
        <pattern id={`${id}-grid`} width="32" height="32" patternUnits="userSpaceOnUse">
          <path
            d="M 32 0 L 0 0 0 32"
            fill="none"
            stroke="#35455f"
            strokeWidth="0.6"
            opacity="0.32"
          />
        </pattern>
      </defs>
      <style>{`
        .eng-motion { transition: transform 650ms cubic-bezier(.22,.7,.3,1), opacity 500ms ease; }
        .eng-stroke { transition: stroke 450ms ease, fill 450ms ease, opacity 450ms ease; }
        @media (prefers-reduced-motion: reduce) {
          .eng-motion, .eng-stroke { transition: none; }
        }
      `}</style>

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

      {/* A cut transit rail visibly opens when the connection is lost. */}
      <g
        className="eng-motion"
        style={{
          transform: disconnected ? "translate(-9px, -2px)" : "translate(0, 0)",
        }}
      >
        <Slab
          x={230}
          y={162}
          w={94}
          d={36}
          z={24}
          fill="#202e40"
          stroke={disconnected ? WARM : EDGE}
          thickness={5}
        />
        <Line
          points={[
            [230, 176, 25],
            [324, 176, 25],
          ]}
          color={disconnected ? WARM : ACCENT}
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
          transform: disconnected ? "translate(9px, 2px)" : "translate(0, 0)",
        }}
      >
        <Slab
          x={336}
          y={162}
          w={99}
          d={36}
          z={24}
          fill="#202e40"
          stroke={disconnected ? WARM : EDGE}
          thickness={5}
        />
        <Line
          points={[
            [336, 176, 25],
            [435, 176, 25],
          ]}
          color={disconnected ? "#65728a" : ACCENT}
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
      <g
        className="eng-motion"
        style={{ opacity: disconnected ? 0 : peerVisible ? 1 : 0.22 }}
      >
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
      <g className="eng-motion" style={{ opacity: disconnected ? 1 : 0 }}>
        <Line
          points={[
            [330, 145, 12],
            [330, 214, 47],
          ]}
          color={WARM}
          dashed
        />
        <Label point={[330, 246, 30]} color={WARM} anchor="middle">
          LINK INTERRUPTED
        </Label>
        <Line
          points={[
            [348, 151, 26],
            [358, 141, 26],
          ]}
          color={WARM}
          width={1.4}
        />
        <Line
          points={[
            [348, 141, 26],
            [358, 151, 26],
          ]}
          color={WARM}
          width={1.4}
        />
      </g>
      <g className="eng-motion" style={{ opacity: disconnected ? 0 : 1 }}>
        <Label point={deployment ? [331, 112, 3] : [331, 250, 28]} anchor="middle">
          UPLINK
        </Label>
      </g>

      <RouteNode x={80} />
      <g className="eng-motion" style={{ opacity: peerVisible ? 1 : 0.18 }}>
        <RouteNode x={435} muted={disconnected} />
      </g>
      <Label point={[98, 266, configured ? 76 : 28]} color={INK}>
        NODE 01
      </Label>
      <Label
        point={[453, 266, configured ? 76 : 28]}
        color={disconnected ? "#8999b5" : INK}
      >
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
          color={disconnected ? WARM : "#8b9fbf"}
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
                transform: `translate(0, ${visible ? -index * (disconnected ? 13 : 7) : 5}px)`,
              }}
            >
              <Slab
                x={96}
                y={17}
                w={107}
                d={48}
                z={11 + index * 2}
                thickness={3}
                fill={disconnected ? "#403d3b" : "#303f55"}
                stroke={disconnected ? WARM : "#8297b7"}
              >
                {[0, 1, 2, 3, 4].map((mark) => (
                  <Line
                    key={mark}
                    points={[
                      [104 + mark * 18, 26, 12 + index * 2],
                      [104 + mark * 18, 54, 12 + index * 2],
                    ]}
                    color={disconnected ? "#b19c7d" : "#6f87aa"}
                  />
                ))}
              </Slab>
            </g>
          );
        })}
        <Label point={[91, -42, -6]} color={disconnected ? WARM : "#acbdd9"}>
          LOCAL BUFFER
        </Label>
      </g>

      {/* Recovery moves a finite group out of the buffer, instead of an endless pulse. */}
      <g
        className="eng-motion"
        style={{
          opacity: catchingUp ? 1 : 0,
          transform: catchingUp ? "translate(0, 0)" : "translate(-35px, -7px)",
        }}
      >
        {[254, 278, 302, 348, 372, 396].map((x) => (
          <Slab
            key={x}
            x={x}
            y={170}
            w={13}
            d={11}
            z={29}
            thickness={3}
            fill="#a8c2e3"
            stroke="#d1dcef"
          />
        ))}
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
        style={{ opacity: disconnected || (deployment && stage === 0) ? 0.25 : 1 }}
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
