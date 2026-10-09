// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Box, DARK, Face, project, Wire } from "./IndustrialGeometry";

const APPLICATIONS = [
  {
    x: 280,
    title: "APPLICATIONS",
    detail: "Operations · Orchestration",
    kind: "application",
  },
  {
    x: 425,
    title: "DATA PLATFORMS",
    detail: "Storage · Analytics",
    kind: "data",
  },
  {
    x: 570,
    title: "MACHINE INTELLIGENCE",
    detail: "Models · Agents",
    kind: "intelligence",
  },
] as const;

const Y = -115;
const Z = 172;

const ApplicationSurface = ({ x }: { x: number }): ReactElement => (
  <g>
    <Wire
      vertices={[
        [x + 15, Y + 17, Z + 5],
        [x + 61, Y + 17, Z + 5],
      ]}
      className="scene-app-ink"
    />
    {[0, 1, 2].map((row) => (
      <g key={row}>
        <Wire
          vertices={[
            [x + 15, Y + 29 + row * 11, Z + 5],
            [x + 39 - row * 4, Y + 29 + row * 11, Z + 5],
          ]}
          className="scene-app-muted"
        />
        <Wire
          vertices={[
            [x + 49, Y + 29 + row * 11, Z + 5],
            [x + 101, Y + 29 + row * 11, Z + 5],
          ]}
          className="scene-app-muted"
        />
      </g>
    ))}
    <Wire
      vertices={[
        [x + 49, Y + 47, Z + 5.5],
        [x + 59, Y + 47, Z + 5.5],
        [x + 66, Y + 41, Z + 5.5],
        [x + 75, Y + 41, Z + 5.5],
        [x + 83, Y + 34, Z + 5.5],
        [x + 91, Y + 36, Z + 5.5],
        [x + 101, Y + 30, Z + 5.5],
      ]}
      className="scene-app-ink"
    />
    <Wire
      vertices={[
        [x + 87, Y + 17, Z + 5],
        [x + 101, Y + 17, Z + 5],
      ]}
      className="scene-app-ink"
    />
  </g>
);

const DataSurface = ({ x }: { x: number }): ReactElement => (
  <g>
    {[0, 1, 2].map((layer) => (
      <Box
        key={layer}
        x={x + 21}
        y={Y + 17}
        z={Z + 5 + layer * 5}
        width={78}
        depth={40}
        height={2}
        material={{
          top: layer === 2 ? "#35485e" : "#283b50",
          left: "#213347",
          right: "#15273a",
        }}
      />
    ))}
    {[0, 1, 2].map((row) => (
      <Wire
        key={row}
        vertices={[
          [x + 32, Y + 26 + row * 10, Z + 17.5],
          [x + 73, Y + 26 + row * 10, Z + 17.5],
        ]}
        className="scene-app-muted"
      />
    ))}
    <Wire
      vertices={[
        [x + 84, Y + 25, Z + 17.5],
        [x + 84, Y + 48, Z + 17.5],
      ]}
      className="scene-app-ink"
    />
  </g>
);

const IntelligenceSurface = ({ x }: { x: number }): ReactElement => (
  <g>
    {[23, 51].map((branch) => (
      <Wire
        key={branch}
        vertices={[
          [x + 27, Y + 37, Z + 5],
          [x + 42, Y + 37, Z + 5],
          [x + 42, Y + branch, Z + 5],
          [x + 61, Y + branch, Z + 5],
          [x + 79, Y + branch, Z + 5],
          [x + 79, Y + 37, Z + 5],
          [x + 96, Y + 37, Z + 5],
        ]}
        className="scene-app-ink"
      />
    ))}
    {[
      [27, 37],
      [61, 23],
      [61, 51],
      [96, 37],
    ].map(([px, py], i) => (
      <g key={i}>
        <Box
          x={x + px - 6}
          y={Y + py - 6}
          z={Z + 5}
          width={12}
          depth={12}
          height={2}
          material={{ top: "#4d6682", left: "#354a61", right: "#25394e" }}
        />
        <Wire
          vertices={[
            [x + px - 3, Y + py, Z + 7.5],
            [x + px + 3, Y + py, Z + 7.5],
          ]}
          className="scene-app-ink"
        />
      </g>
    ))}
  </g>
);

export const IndustrialSoftware = (): ReactElement => (
  <g className="scene-applications">
    {APPLICATIONS.map(({ x, title, detail, kind }) => {
      const [labelX, labelY] = project([x + 60, Y + 24, Z + 58]);
      return (
        <g key={kind} className="scene-software-module" data-kind={kind}>
          <Box x={x} y={Y} z={Z} width={120} depth={74} height={4} material={DARK} />
          <Face
            vertices={[
              [x + 7, Y + 7, Z + 4.5],
              [x + 113, Y + 7, Z + 4.5],
              [x + 113, Y + 66, Z + 4.5],
              [x + 7, Y + 66, Z + 4.5],
            ]}
            fill="#162639"
            className="scene-app-surface"
          />
          {kind === "application" && <ApplicationSurface x={x} />}
          {kind === "data" && <DataSurface x={x} />}
          {kind === "intelligence" && <IntelligenceSurface x={x} />}
          <text x={labelX} y={labelY} textAnchor="middle" className="scene-app-label">
            {title}
          </text>
          <text
            x={labelX}
            y={labelY + 18}
            textAnchor="middle"
            className="scene-app-detail"
          >
            {detail}
          </text>
        </g>
      );
    })}
  </g>
);
