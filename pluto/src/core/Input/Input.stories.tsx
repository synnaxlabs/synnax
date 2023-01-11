// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import type { ComponentMeta } from "@storybook/react";
import { TimeStamp } from "@synnaxlabs/client";

import { Text } from "../Typography";

import { Input } from ".";

const story: ComponentMeta<typeof Input> = {
  title: "Core/Input",
  component: Input,
};

export const Basic = (): JSX.Element => {
  const [value, setValue] = useState("");
  return <Input value={value} onChange={setValue} />;
};

export const Time = (): JSX.Element => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  return (
    <>
      <Input.Time value={value} onChange={setValue} />;
      <Text.DateTime level="h1" format="dateTime">
        {value}
      </Text.DateTime>
    </>
  );
};

export const Date = (): JSX.Element => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  console.log(value);
  return (
    <>
      <Input.Date value={value} onChange={setValue} />;
      <Text.DateTime level="h1" format="dateTime">
        {value}
      </Text.DateTime>
    </>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
