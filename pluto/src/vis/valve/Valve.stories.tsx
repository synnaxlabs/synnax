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

import { Control } from "@/telem/control";
import { Controller } from "@/telem/control/Controller";
import { Canvas } from "@/vis/canvas";
import { Valve } from "@/vis/valve";

import { Bool } from "@/telem/bool";
import { Remote } from "@/telem/remote";

const story: Meta<typeof Valve.Valve> = {
  title: "Valve",
  component: Valve.Valve,
};

const Example = (): ReactElement => {
  const numericSource = Remote.useNumericSource({
    channel: 65542,
  });
  const booleanSource = Bool.withinBounds({
    wrap: numericSource,
    trueBound: { lower: 30, upper: 40 },
  });
  const numericSink = Control.useNumericSink({
    channel: 65541,
  });
  const booleanSink = Bool.setpoint({
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
      <Controller authority={5} acquireTrigger={0} name="Controller">
        <Valve.Valve color="#fc3d03" source={booleanSource} sink={booleanSink} />
      </Controller>
    </Canvas.Canvas>
  );
};

export const Primary: StoryFn<typeof Valve> = () => <Example />;

export default story;
