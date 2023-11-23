// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Canvas } from "@/vis/canvas";
import { PID } from "@/vis/pid";

import {
  AngledValve,
  FourWayValve,
  Pump,
  Regulator,
  SolenoidValve,
  ThreeWayValve,
  Valve,
} from "./symbols/Symbols";

const story: Meta<typeof PID.PID> = {
  title: "PID",
  component: PID.PID,
};

const Example = (): ReactElement => {
  const props = PID.use({
    initialNodes: [
      {
        key: "1",
        position: {
          x: 100,
          y: 100,
        },
        data: {},
      },
      {
        key: "2",
        position: {
          x: 500,
          y: 200,
        },
      },
    ],
    initialEdges: [
      {
        key: "dog",
        source: "1",
        target: "2",
        points: [],
      },
    ],
  });

  const S = useCallback(
    (p) => (
      <AngledValve {...p} color="black" orientation="top" label="Label" units="psi" />
    ),
    [],
  );

  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
      }}
    >
      <PID.PID {...props} editable>
        <PID.NodeRenderer>{S}</PID.NodeRenderer>
      </PID.PID>
    </Canvas.Canvas>
  );
};

export const Primary: StoryFn<typeof PID> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
