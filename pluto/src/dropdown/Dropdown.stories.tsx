// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Button } from "@/button";
import { Dropdown } from "@/dropdown";

const story: Meta<typeof Dropdown.DialogProps> = {
  title: "Dropdown",
  component: Dropdown.Dialog,
};

export const Primary: StoryFn<typeof Dropdown.Dialog> = () => {
  return (
    <Dropdown.Dialog visible style={{ position: "fixed", bottom: 100 }}>
      <Button.Button>Hello</Button.Button>
      <h1>Content</h1>
    </Dropdown.Dialog>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
