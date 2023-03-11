// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Tag } from ".";

const story: ComponentMeta<typeof Tag> = {
  title: "Core/Tag",
  component: Tag,
};

const Template: ComponentStory<typeof Tag> = (props) => <Tag {...props} />;

export const Primary: ComponentStory<typeof Tag> = Template.bind({});
Primary.args = {
  children: "Tag",
  onClose: () => undefined,
  variant: "outlined",
  size: "small",
};

// eslint-disable-next-line import/no-default-export
export default story;
