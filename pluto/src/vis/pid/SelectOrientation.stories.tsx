// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Align } from "@/align";
import { Pluto } from "@/pluto";

import { SelectOrientation, type OrientationValue } from "./SelectOrientation";
import { FourWayValve, ReliefValve, ThreeWayValve } from "./Symbols";

const story: Meta<typeof SelectOrientation> = {
  title: "OrientationControl",
  component: SelectOrientation,
};

const Example = (): ReactElement => {
  const [value, setValue] = useState<OrientationValue>({
    inner: "top",
    outer: "left",
  });

  return (
    <Align.Space align="start">
      <SelectOrientation value={value} onChange={setValue} />;
    </Align.Space>
  );
};

export const Primary: StoryFn<typeof SelectOrientation> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
