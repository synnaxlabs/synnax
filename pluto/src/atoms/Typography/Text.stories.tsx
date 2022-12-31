// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiOutlineDelete } from "react-icons/ai";

import { Text } from ".";

const story: ComponentMeta<typeof Text> = {
  title: "Atoms/Text",
  component: Text,
};

export const Basic: ComponentStory<typeof Text> = () => <Text level="h2">Hello</Text>;

export const WithIcon: ComponentStory<typeof Text> = () => (
  <Text.WithIcon
    startIcon={<AiOutlineDelete />}
    endIcon={<AiFillDatabase />}
    level="h2"
    divided
  >
    Text
  </Text.WithIcon>
);

export const Editable: ComponentStory<typeof Text> = () => {
  const [text, setText] = useState("My Text");
  return (
    <>
      <Text.Editable level="h1" onChange={setText}>
        {text}
      </Text.Editable>
      <Text level="h5">{text}</Text>
    </>
  );
};

export default story;
