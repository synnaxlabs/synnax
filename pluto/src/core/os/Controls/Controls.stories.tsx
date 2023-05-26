// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Controls } from ".";

const story: Meta<typeof Controls> = {
  title: "OS/Controls",
  component: Controls,
  argTypes: {},
};

export const MacOS: StoryFn<typeof Controls> = () => <Controls forceOS="MacOS" />;

export const Windows: StoryFn<typeof Controls> = () => <Controls forceOS="Windows" />;

// eslint-disable-next-line import/no-default-export
export default story;
