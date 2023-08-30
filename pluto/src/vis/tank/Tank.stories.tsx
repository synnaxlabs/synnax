// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Meta, StoryFn } from "@storybook/react";

import { Tank } from "@/vis/tank";

const story: Meta<typeof Tank.Tank> = {
  title: "Tank",
  component: Tank.Tank,
};

const Example = (): ReactElement => {
  return <Tank.Tank dimensions={{ width: 500, height: 200 }} />;
};

export const Primary: StoryFn<typeof Tank> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
