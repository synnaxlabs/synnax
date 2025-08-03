// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { Tag } from "@/tag";

import { PADDING_STYLE } from "./constants";

export const TagShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Flex.Box y>
        <Tag.Tag size="huge">Hello</Tag.Tag>
        <Tag.Tag size="large">Hello</Tag.Tag>
        <Tag.Tag size="medium">Hello</Tag.Tag>
        <Tag.Tag size="small">Hello</Tag.Tag>
        <Tag.Tag size="tiny">Hello</Tag.Tag>
      </Flex.Box>
      <Flex.Box y>
        <Tag.Tag size="huge" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="large" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="medium" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="small" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="tiny" color="#00E3E2">
          Hello
        </Tag.Tag>
      </Flex.Box>
      <Flex.Box y>
        <Tag.Tag size="huge" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="large" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="medium" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="small" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="tiny" onClose={console.log}>
          Hello
        </Tag.Tag>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);
