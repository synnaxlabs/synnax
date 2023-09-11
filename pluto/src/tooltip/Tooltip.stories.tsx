// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";
import { location } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Button } from "@/button";
import { Tooltip } from "@/tooltip";

const story: Meta<typeof Tooltip.Dialog> = {
  title: "Tooltip",
  component: Tooltip.Dialog,
};

const Template = (): ReactElement => (
  <Align.Center style={{ width: "100%", height: "100%" }}>
    <Tooltip.Dialog location={location.TOP_RIGHT}>
      <p>Top Right - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.TOP_LEFT}>
      <p>Top left - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.BOTTOM_RIGHT}>
      <p>Bottom Right - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.BOTTOM_LEFT}>
      <p>Bottom Left - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.BOTTOM_CENTER}>
      <p>Bottom Center - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.LEFT_CENTER}>
      <p>Left Center - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.RIGHT_CENTER}>
      <p>Right Center - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
    <Tooltip.Dialog location={location.TOP_CENTER}>
      <p>Top Center - Super Cool Stuff</p>
      <Button.Button>Button</Button.Button>
    </Tooltip.Dialog>
  </Align.Center>
);

export const Primary: StoryFn<typeof Tooltip.Dialog> = Template.bind({});

// eslint-disable-next-line import/no-default-export
export default story;
