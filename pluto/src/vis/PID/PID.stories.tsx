// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { AetherNoopTelem } from "@/telem/noop/noop";
import { Canvas } from "@/vis/canvas/Canvas";
import { PID } from "@/vis/pid";
import { ValuePIDElementSpec } from "@/visupper";
import { ValvePIDElementSpec } from "@/visupper/PID/ValvePIDElement";

const story: Meta<typeof PID> = {
  title: "Core/Vis/PID",
  component: PID,
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
  return (
    <Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
      }}
    >
      {/* <div style={{ height: "50%" }} /> */}
      <PID {...props}>
        {(props) => (
          <ValuePIDElementSpec.Element {...props} label="Label" units="psi" />
        )}
      </PID>
    </Canvas>
  );
};

export const Primary: StoryFn<typeof PID> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
