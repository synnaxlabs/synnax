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
import { ReactFlowProvider } from "reactflow";

import { Align } from "@/align";
import { Primitives } from "@/vis/pid/symbols/primitives";

const story: Meta<typeof Primitives.Tank> = {
  title: "Primitive Symbols",
  component: Primitives.Tank,
};

const COLOR = "#000000";

const Example = (): ReactElement => {
  return (
    <Align.Space className="story">
      <ReactFlowProvider>
        <Align.Space direction="x" wrap size="large">
          <Primitives.SolenoidValve color={COLOR} normallyOpen enabled />
          <Primitives.SolenoidValve color={COLOR} normallyOpen />
          <Primitives.SolenoidValve color={COLOR} enabled />
          <Primitives.SolenoidValve color={COLOR} />
          <Primitives.SolenoidValve color={COLOR} />
          <Primitives.SolenoidValve color={COLOR} orientation="left" />
          <Primitives.SolenoidValve color={COLOR} orientation="right" />
          <Primitives.SolenoidValve color={COLOR} orientation="top" />
          <Primitives.SolenoidValve color={COLOR} orientation="bottom" />
          <Primitives.Valve color={COLOR} enabled />
          <Primitives.Valve color={COLOR} />
          <Primitives.Valve color={COLOR} orientation="left" />
          <Primitives.Valve color={COLOR} orientation="right" />
          <Primitives.Valve color={COLOR} orientation="top" />
          <Primitives.Valve color={COLOR} orientation="bottom" />
          <Primitives.ThreeWayValve color={COLOR} />
          <Primitives.ThreeWayValve color={COLOR} enabled />
          <Primitives.ThreeWayValve color={COLOR} orientation="left" />
          <Primitives.ThreeWayValve color={COLOR} orientation="right" />
          <Primitives.ThreeWayValve color={COLOR} orientation="top" />
          <Primitives.ThreeWayValve color={COLOR} orientation="bottom" />
          <Primitives.FourWayValve color={COLOR} />
          <Primitives.FourWayValve color={COLOR} enabled />
          <Primitives.FourWayValve color={COLOR} orientation="left" />
          <Primitives.FourWayValve color={COLOR} orientation="right" />
          <Primitives.FourWayValve color={COLOR} orientation="top" />
          <Primitives.FourWayValve color={COLOR} orientation="bottom" />
          <Primitives.AngledValve color={COLOR} />
          <Primitives.AngledValve color={COLOR} enabled />
          <Primitives.AngledValve color={COLOR} orientation="left" />
          <Primitives.AngledValve color={COLOR} orientation="right" />
          <Primitives.AngledValve color={COLOR} orientation="top" />
          <Primitives.AngledValve color={COLOR} orientation="bottom" />
          <Primitives.Pump color={COLOR} />
          <Primitives.Pump color={COLOR} enabled />
          <Primitives.Pump color={COLOR} orientation="left" />
          <Primitives.Pump color={COLOR} orientation="right" />
          <Primitives.Pump color={COLOR} orientation="top" />
          <Primitives.Pump color={COLOR} orientation="bottom" />
          <Primitives.BurstDisc color={COLOR} />
          <Primitives.BurstDisc color={COLOR} orientation="left" />
          <Primitives.BurstDisc color={COLOR} orientation="right" />
          <Primitives.BurstDisc color={COLOR} orientation="top" />
          <Primitives.BurstDisc color={COLOR} orientation="bottom" />
          <Primitives.Cap color={COLOR} />
          <Primitives.Cap color={COLOR} orientation="left" />
          <Primitives.Cap color={COLOR} orientation="right" />
          <Primitives.Cap color={COLOR} orientation="top" />
          <Primitives.Cap color={COLOR} orientation="bottom" />
          <Primitives.Regulator color={COLOR} />
          <Primitives.Regulator color={COLOR} orientation="left" />
          <Primitives.Regulator color={COLOR} orientation="right" />
          <Primitives.Regulator color={COLOR} orientation="top" />
          <Primitives.Regulator color={COLOR} orientation="bottom" />
          <Primitives.Orifice color={COLOR} />
          <Primitives.Orifice color={COLOR} orientation="left" />
          <Primitives.Orifice color={COLOR} orientation="right" />
          <Primitives.Orifice color={COLOR} orientation="top" />
          <Primitives.Orifice color={COLOR} orientation="bottom" />
          <Primitives.Filter color={COLOR} />
          <Primitives.Filter color={COLOR} orientation="left" />
          <Primitives.Filter color={COLOR} orientation="right" />
          <Primitives.Filter color={COLOR} orientation="top" />
          <Primitives.Filter color={COLOR} orientation="bottom" />
          <Primitives.NeedleValve color={COLOR} />
          <Primitives.NeedleValve color={COLOR} orientation="left" />
          <Primitives.NeedleValve color={COLOR} orientation="right" />
          <Primitives.NeedleValve color={COLOR} orientation="top" />
          <Primitives.NeedleValve color={COLOR} orientation="bottom" />
          <Primitives.CheckValve color={COLOR} />
          <Primitives.CheckValve color={COLOR} orientation="left" />
          <Primitives.CheckValve color={COLOR} orientation="right" />
          <Primitives.CheckValve color={COLOR} orientation="top" />
          <Primitives.CheckValve color={COLOR} orientation="bottom" />
          <Primitives.ManualValve color={COLOR} />
          <Primitives.ManualValve color={COLOR} orientation="left" />
          <Primitives.ManualValve color={COLOR} orientation="right" />
          <Primitives.ManualValve color={COLOR} orientation="top" />
          <Primitives.ManualValve color={COLOR} orientation="bottom" />
          <Primitives.Tank
            dimensions={{ width: 300, height: 500 }}
            borderRadius={{ x: 50, y: 10 }}
            color={COLOR}
          />
          <Primitives.AngledReliefValve color={COLOR} />
          <Primitives.AngledReliefValve color={COLOR} orientation="left" />
          <Primitives.AngledReliefValve color={COLOR} orientation="right" />
          <Primitives.AngledReliefValve color={COLOR} orientation="top" />
          <Primitives.AngledReliefValve color={COLOR} orientation="bottom" />
          <Primitives.Switch color={COLOR} orientation="left" />
        </Align.Space>
      </ReactFlowProvider>
    </Align.Space>
  );
};

export const Primary: StoryFn<typeof Primitives.Tank> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
