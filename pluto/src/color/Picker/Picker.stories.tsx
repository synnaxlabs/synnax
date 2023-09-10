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

import { type Accordion } from "@/accordion";
import { Color } from "@/color";
import { Picker } from "@/color/Picker";

const story: Meta<typeof Picker> = {
  title: "Color.Picker",
  component: Picker,
};

export const Primary: StoryFn<typeof Accordion> = () => {
  const [value, setValue] = useState<Color.Color>(Color.ZERO.setAlpha(1));
  return <Picker value={value} onChange={setValue} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
