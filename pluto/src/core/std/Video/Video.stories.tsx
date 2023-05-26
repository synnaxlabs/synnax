// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Video, VideoProps } from "@/core/std/Video";

const story: Meta<typeof Video> = {
  title: "Core/Video",
  component: Video,
  argTypes: {},
};

const Template: StoryFn<typeof Video> = (args: VideoProps) => <Video {...args} />;

export const Default: StoryFn<typeof Video> = Template.bind({});
Default.args = {
  href: "https://www.w3schools.com/html/mov_bbb.mp4",
  autoPlay: true,
  loop: true,
};

// eslint-disable-next-line import/no-default-export
export default story;
