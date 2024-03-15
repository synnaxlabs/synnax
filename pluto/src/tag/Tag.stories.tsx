// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Tag } from "@/tag";

const story: Meta<typeof Tag.Tag> = {
  title: "Tag",
  component: Tag.Tag,
};

const Template: StoryFn<typeof Tag.Tag> = (props) => <Tag.Tag {...props} />;

export const Primary: StoryFn<typeof Tag.Tag> = Template.bind({});
Primary.args = {
  children: "Tag",
  onClose: () => undefined,
  variant: "outlined",
  color: "var(--pluto-primary-z)",
  size: "small",
};

// eslint-disable-next-line import/no-default-export
export default story;
