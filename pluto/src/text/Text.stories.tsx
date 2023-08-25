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
import { AiFillDatabase, AiOutlineDelete } from "react-icons/ai";

import { Text } from "@/text";

const story: Meta<typeof Text.Text> = {
  title: "Text",
  component: Text.Text,
};

export const Basic: StoryFn<typeof Text> = () => (
  <Text.Text level="h2">Hello</Text.Text>
);

export const WithIcon: StoryFn<typeof Text> = () => (
  <Text.WithIcon
    startIcon={<AiOutlineDelete />}
    endIcon={<AiFillDatabase />}
    level="h2"
    divided
  >
    Text
  </Text.WithIcon>
);

export const Editable: StoryFn<typeof Text> = () => {
  const [text, setText] = useState("My Text");
  return (
    <>
      <Text.Editable level="h1" onChange={setText} value={text} />
      <Text.Text level="h5">{text}</Text.Text>
    </>
  );
};

export const Keyboard: StoryFn<typeof Text> = () => (
  <Text.Keyboard level="p">⌘</Text.Keyboard>
);

// eslint-disable-next-line import/no-default-export
export default story;
