// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentMeta } from "@storybook/react";

import { Rule, RuleAnnotationProps } from ".";

const story: ComponentMeta<typeof Rule> = {
  title: "Visualization/Rule",
  component: Rule,
};

const annotations: RuleAnnotationProps[] = [
  {
    key: "1",
    position: 100,
    values: {
      "ec.pressure[7]": "592.2",
    },
  },
];

export const Default = (): JSX.Element => (
  <svg width="100%" height="100%">
    <Rule
      position={200}
      direction="y"
      size={{ lower: 0, upper: 500 }}
      annotations={annotations}
    />
  </svg>
);

// eslint-disable-next-line import/no-default-export
export default story;
