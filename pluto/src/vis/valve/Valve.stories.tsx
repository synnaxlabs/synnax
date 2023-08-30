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

import { Bool } from "@/telem/bool";
import { Control } from "@/telem/control";
import { Controller } from "@/telem/control/Controller";
import { Remote } from "@/telem/remote";
import { Canvas } from "@/vis/canvas";
import { Valve } from "@/vis/valve";

const story: Meta<typeof Valve.Valve> = {
  title: "Valve",
  component: Valve.Valve,
};

const Example = (): ReactElement => {
  const numericSource = Remote.useNumericSource({
    channel: 65542,
  });
  const booleanSource = Bool.useNumericConverterSource({
    wrap: numericSource,
    trueBound: new Bounds(30, 40),
  });
  const numericSink = Control.useNumeric({
    channel: 65541,
  });
  const booleanSink = Bool.useNumericConverterSink({
    wrap: numericSink,
    truthy: 1,
    falsy: 0,
  });

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
      <Controller authority={5}>
        <Valve.Valve color="#fc3d03" source={booleanSource} sink={booleanSink} />
      </Controller>
    </Canvas.Canvas>
  );
};

export const Primary: StoryFn<typeof Valve> = () => <Example />;

export default story;
