// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

import { PADDING_STYLE } from "./constants";

export const TextShowcase = () => (
  <Flex.Box x bordered style={PADDING_STYLE} rounded={1} gap="large">
    <Flex.Box y>
      <Text.Text level="h1">Hello</Text.Text>
      <Text.Text level="h2">Hello</Text.Text>
      <Text.Text level="h3">Hello</Text.Text>
      <Text.Text level="h4">Hello</Text.Text>
      <Text.Text level="h5">Hello</Text.Text>
      <Text.Text>Hello</Text.Text>
      <Text.Text level="small">Hello</Text.Text>
    </Flex.Box>
    <Flex.Box y>
      <Text.Text>Hello</Text.Text>
      <Text.Text color={10}>Hello</Text.Text>
      <Text.Text color={9}>Hello</Text.Text>
      <Text.Text color={8}>Hello</Text.Text>
      <Text.Text color={7}>Hello</Text.Text>
      <Text.Text color={6}>Hello</Text.Text>
      <Text.Text color={5}>Hello</Text.Text>
      <Text.Text color={4}>Hello</Text.Text>
      <Text.Text color={3}>Hello</Text.Text>
      <Text.Text color={2}>Hello</Text.Text>
      <Text.Text color={1}>Hello</Text.Text>
    </Flex.Box>
    <Flex.Box y>
      <Text.Text href="https://www.google.com" level="h1">
        Hello
      </Text.Text>
      <Text.Text href="https://www.google.com" level="h2">
        Hello
      </Text.Text>
      <Text.Text href="https://www.google.com" level="h3">
        Hello
      </Text.Text>
      <Text.Text href="https://www.google.com" level="h4">
        Hello
      </Text.Text>
      <Text.Text href="https://www.google.com" level="h5">
        Hello
      </Text.Text>
      <Text.Text href="https://www.google.com">Hello</Text.Text>
      <Text.Text href="https://www.google.com" level="small">
        Hello
      </Text.Text>
    </Flex.Box>
    <Flex.Box y>
      <Text.Text variant="code">Hello</Text.Text>
      <Flex.Box x gap="small">
        <Text.Text variant="keyboard">
          <Text.Symbols.Meta />
        </Text.Text>
        <Text.Text variant="keyboard">
          <Text.Symbols.Alt />
        </Text.Text>
      </Flex.Box>
      <Triggers.Text trigger={["Control", "Enter"]}>Hello</Triggers.Text>
    </Flex.Box>
  </Flex.Box>
);
