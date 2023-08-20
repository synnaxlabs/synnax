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
import { XYLocation } from "@synnaxlabs/x";

import { Button } from "../button";
import { Space } from "../align";

import { Tooltip } from "@/tooltip";

const story: Meta<typeof Tooltip> = {
  title: "Core/Standard/Tooltip",
  component: Tooltip,
};

const Template = (): ReactElement => (
  <Space.Centered style={{ width: "100%", height: "100%" }}>
    <Tooltip location={XYLocation.TOP_RIGHT.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.TOP_LEFT.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.BOTTOM_RIGHT.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.BOTTOM_LEFT.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.BOTTOM_CENTER.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.LEFT_CENTER.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.RIGHT_CENTER.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
    <Tooltip location={XYLocation.TOP_CENTER.crude}>
      <p>Tooltip Conjent</p>
      <Button>Button</Button>
    </Tooltip>
  </Space.Centered>
);

export const Primary: StoryFn<typeof Tooltip> = Template.bind({});

// eslint-disable-next-line import/no-default-export
export default story;
