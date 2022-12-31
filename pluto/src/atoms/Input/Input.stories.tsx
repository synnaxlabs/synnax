// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta } from "@storybook/react";

import { Input } from ".";

const story: ComponentMeta<typeof Input> = {
  title: "Atoms/Input",
  component: Input,
};

export const Basic = (): JSX.Element => <Input />;

export const Time = (): JSX.Element => (
  <Input.Time size="medium" onChange={console.log} />
);

export const Date = (): JSX.Element => (
  <Input.Date size="medium" onChange={console.log} />
);

export default story;
