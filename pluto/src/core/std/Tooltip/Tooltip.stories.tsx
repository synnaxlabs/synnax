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

import { Button } from "../Button";

import { Tooltip } from "@/core/std/Tooltip";

const story: Meta<typeof Tooltip> = {
  title: "Core/Standard/Tooltip",
  component: Tooltip,
};

const Template = (): ReactElement => (
  <Tooltip>
    <p>Tooltip Content</p>
    <Button>Button</Button>
  </Tooltip>
);

export const Primary: StoryFn<typeof Tooltip> = Template.bind({});

// eslint-disable-next-line import/no-default-export
export default story;
