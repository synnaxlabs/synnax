// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Meta, type StoryFn } from "@storybook/react";

import { telem } from "@/telem/aether";
import { Canvas } from "@/vis/canvas";
import { Value } from "@/vis/pid/symbols/Symbols";

const story: Meta<typeof Value> = {
  title: "Value",
  component: Value,
};

const Example = (): ReactElement => {
  const t = telem.fixedString("120 PSI");
  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <Value
        orientation="bottom"
        label={{
          orientation: "top",
          label: "Hello",
          level: "small",
        }}
        level="small"
        color="#000000"
        telem={telem.fixedString("120 PSI")}
      />
    </Canvas.Canvas>
  );
};

export const Primary: StoryFn<typeof Value> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
