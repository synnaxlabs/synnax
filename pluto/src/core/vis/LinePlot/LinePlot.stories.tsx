// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Line } from "../Line";

import { LinePlot } from "@/core/vis/LinePlot";
import { Pluto } from "@/Pluto";
import { useStaticTelem } from "@/telem/staticTelem";

const story: Meta<typeof LinePlot> = {
  title: "Vis/LinePlot",
  component: LinePlot,
};

const Example = (): ReactElement => {
  const telem = useStaticTelem({
    x: [new Int32Array([1, 2, 3])],
    y: [new Int32Array([1, 2, 3])],
  });
  return (
    <Pluto>
      <LinePlot>
        <LinePlot.XAxis label="Time" color="#FFFFFF">
          <LinePlot.YAxis label="PSI" color="#FFFFFF">
            <Line telem={telem} color="#FFFFFF" />
          </LinePlot.YAxis>
        </LinePlot.XAxis>
      </LinePlot>
    </Pluto>
  );
};

export const Primary: StoryFn<typeof LinePlot> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
