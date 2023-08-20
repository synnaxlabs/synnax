// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import type { Meta } from "@storybook/react";
import { TimeStamp } from "@synnaxlabs/x";

import { Input } from "@/input";
import { Text } from "@/text";

const story: Meta<typeof Input> = {
  title: "Core/Standard/Input",
  component: Input,
};

export const Basic = (): ReactElement => {
  const [value, setValue] = useState("");
  return <Input value={value} onChange={setValue} />;
};

export const Time = (): ReactElement => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  return (
    <>
      <Input.Time value={value} onChange={setValue} />
      <Text.DateTime level="h1" format="time" suppliedTZ="UTC" displayTZ="local">
        {value}
      </Text.DateTime>
      <h1>{new TimeStamp(value).fString("ISO", "UTC")}</h1>
    </>
  );
};

export const Date = (): ReactElement => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  return (
    <>
      <Input.Date value={value} onChange={setValue} />;
      <Text.DateTime level="h1" format="dateTime" suppliedTZ="UTC" displayTZ="UTC">
        {value}
      </Text.DateTime>
      <h1>{new TimeStamp(value).fString("ISO", "UTC")}</h1>
    </>
  );
};

export const Number = (): ReactElement => {
  const [value, setValue] = useState<number>(0);
  return <Input.Number value={value} onChange={setValue} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
