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

import { SubcategorySection } from "./SubcategorySection";

export const DividerShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Vertical Dividers"
        description="Vertical separators for dividing content horizontally within flex containers"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Between Text Elements
          </Text.Text>
          <Flex.Box
            x
            style={{
              padding: "2rem",
              border: "1px solid var(--pluto-gray-l1)",
              borderRadius: "8px",
              background: "var(--pluto-gray-l0)",
            }}
          >
            <Text.Text>Before</Text.Text>
            <Divider.Divider />
            <Text.Text>After</Text.Text>
          </Flex.Box>
          <Flex.Box
            x
            style={{
              padding: "2rem",
              border: "1px solid var(--pluto-gray-l1)",
              borderRadius: "8px",
              background: "var(--pluto-gray-l0)",
            }}
          >
            <Text.Text>Left</Text.Text>
            <Divider.Divider />
            <Text.Text>Center</Text.Text>
            <Divider.Divider />
            <Text.Text>Right</Text.Text>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Horizontal Dividers"
        description="Horizontal separators for dividing content vertically within flex containers"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Between Text Elements
          </Text.Text>
          <Flex.Box
            y
            style={{
              padding: "2rem",
              border: "1px solid var(--pluto-gray-l1)",
              borderRadius: "8px",
              background: "var(--pluto-gray-l0)",
            }}
          >
            <Text.Text>Above</Text.Text>
            <Divider.Divider x />
            <Text.Text>Below</Text.Text>
          </Flex.Box>
          <Flex.Box
            y
            style={{
              padding: "2rem",
              border: "1px solid var(--pluto-gray-l1)",
              borderRadius: "8px",
              background: "var(--pluto-gray-l0)",
            }}
          >
            <Text.Text>Top</Text.Text>
            <Divider.Divider x />
            <Text.Text>Middle</Text.Text>
            <Divider.Divider x />
            <Text.Text>Bottom</Text.Text>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
