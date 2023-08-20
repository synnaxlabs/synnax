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

const story: Meta<typeof Tag> = {
  title: "Core/Standard/Tag",
  component: Tag,
};

const Template: StoryFn<typeof Tag> = (props) => <Tag {...props} />;

export const Primary: StoryFn<typeof Tag> = Template.bind({});
Primary.args = {
  children: "Tag",
  onClose: () => undefined,
  variant: "outlined",
  size: "small",
};

// eslint-disable-next-line import/no-default-export
export default story;
