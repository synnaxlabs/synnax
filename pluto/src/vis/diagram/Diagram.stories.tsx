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
import { Diagram } from "@/vis/diagram";
import { Primitives } from "@/vis/pid/primitives";

const story: Meta<typeof Diagram.Diagram> = {
  title: "Diagram",
  component: Diagram.Diagram,
};

const Example = (): ReactElement => {
  const props = Diagram.use({
    initialNodes: [
      {
        key: "1",
        position: {
          x: 0,
          y: 0,
        },
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
      // {
      //   key: "dog",
      //   source: "1",
      //   target: "2",
      // },
    ],
  });

  const S = useCallback(() => <Primitives.Switch />, []);

  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
      }}
    >
      <Diagram.Diagram {...props} editable>
        <Diagram.NodeRenderer>{S}</Diagram.NodeRenderer>
      </Diagram.Diagram>
    </Canvas.Canvas>
  );
};

export const Primary: StoryFn<typeof Diagram.Diagram> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
