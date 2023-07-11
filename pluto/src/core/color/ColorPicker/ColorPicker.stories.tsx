// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { ColorPicker } from "./ColorPicker";

import { Color } from "@/core/color/color";
import { Accordion } from "@/core/std/Accordion";

const story: Meta<typeof ColorPicker> = {
  title: "Color/Picker",
  component: ColorPicker,
};

export const Primary: StoryFn<typeof Accordion> = () => {
  const [value, setValue] = useState<Color>(Color.ZERO.setAlpha(1));
  return <ColorPicker value={value} onChange={setValue} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
