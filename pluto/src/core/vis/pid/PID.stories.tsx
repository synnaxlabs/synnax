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

import { Text } from "@/core/std";
import { Canvas } from "@/core/vis/Canvas/Canvas";
import { PID } from "@/core/vis/PID";

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
      },
      {
        key: "2",
        position: {
          x: 200,
          y: 200,
        },
      },
    ],
    initialEdges: [],
  });
  const [v, setV] = useState("dog");
  return (
    <Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
      }}
    >
      <PID {...props}>
        {({ elementKey: key, selected }) => {
          return <Text.Editable value={v} onChange={setV} level="h4" />;
        }}
      </PID>
    </Canvas>
  );
};

export const Primary: StoryFn<typeof PID> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
