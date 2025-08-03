// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Text } from "@/text";

import { PADDING_STYLE } from "./constants";

export const DividerShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Text.Text>Before</Text.Text>
      <Divider.Divider />
      <Text.Text>After</Text.Text>
    </Flex.Box>
    <Flex.Box y>
      <Text.Text>Above</Text.Text>
      <Divider.Divider x />
      <Text.Text>Below</Text.Text>
    </Flex.Box>
    <Flex.Box x>
      <Text.Text>Left</Text.Text>
      <Divider.Divider />
      <Text.Text>Right</Text.Text>
    </Flex.Box>
  </Flex.Box>
);
