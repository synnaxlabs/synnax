// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Meta, StoryFn } from "@storybook/react";
import { Bounds } from "@synnaxlabs/x";

import { Controller } from "@/telem/control/Controller";

import { Canvas } from "../canvas/Canvas";

import { Valve } from "./Valve";

import { Telem } from "@/telem";

const story: Meta<typeof Valve> = {
  title: "Core/Vis/Valve",
  component: Valve,
};

const Example = (): ReactElement => {
  const numericSource = Telem.Remote.useNumeric({
    channel: 65542,
  });
  const booleanSource = Telem.Boolean.useNumericConverterSource({
    wrap: numericSource,
    trueBound: new Bounds(30, 40),
  });
  const numericSink = Telem.Control.useNumeric({
    channel: 65541,
  });
  const booleanSink = Telem.Boolean.useNumericConverterSink({
    wrap: numericSink,
    truthy: 1,
    falsy: 0,
  });

  return (
    <Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <Controller authority={5}>
        <Valve color="#fc3d03" source={booleanSource} sink={booleanSink} />
      </Controller>
    </Canvas>
  );
};

export const Primary: StoryFn<typeof Valve> = () => <Example />;

export default story;
