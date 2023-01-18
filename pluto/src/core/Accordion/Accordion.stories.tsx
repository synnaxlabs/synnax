// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Accordion, AccordionEntry } from ".";

const story: ComponentMeta<typeof Accordion> = {
  title: "Core/Accordion",
  component: Accordion,
};

const data: AccordionEntry[] = [
  {
    key: "cluster",
    name: "Cluster",
    content: <p>Content</p>,
  },
  {
    key: "Devices",
    name: "Devices",
    content: <p>Content</p>,
  },
];

export const Primary: ComponentStory<typeof Accordion> = () => (
  <Accordion data={data} />
);

// eslint-disable-next-line import/no-default-export
export default story;
