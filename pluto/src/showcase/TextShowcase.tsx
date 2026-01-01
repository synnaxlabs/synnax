// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

import { SubcategorySection } from "./SubcategorySection";

export const TextShowcase = () => (
  <Flex.Box y pack empty rounded>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Typography Hierarchy"
        description="Text levels from largest heading (h1) to smallest body text, showing proper hierarchy and sizing"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Text Levels
          </Text.Text>
          <Flex.Box y>
            <Text.Text level="h1">Hello (H1)</Text.Text>
            <Text.Text level="h2">Hello (H2)</Text.Text>
            <Text.Text level="h3">Hello (H3)</Text.Text>
            <Text.Text level="h4">Hello (H4)</Text.Text>
            <Text.Text level="h5">Hello (H5)</Text.Text>
            <Text.Text>Hello (Body)</Text.Text>
            <Text.Text level="small">Hello (Small)</Text.Text>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
      <SubcategorySection
        title="Color Variations"
        description="Text color intensity levels from highest (default) to lowest contrast"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Color Levels (10 = highest contrast, 1 = lowest)
          </Text.Text>
          <Flex.Box y>
            <Text.Text>Hello (Default)</Text.Text>
            <Text.Text color={10}>Hello (10)</Text.Text>
            <Text.Text color={9}>Hello (9)</Text.Text>
            <Text.Text color={8}>Hello (8)</Text.Text>
            <Text.Text color={7}>Hello (7)</Text.Text>
            <Text.Text color={6}>Hello (6)</Text.Text>
            <Text.Text color={5}>Hello (5)</Text.Text>
            <Text.Text color={4}>Hello (4)</Text.Text>
            <Text.Text color={3}>Hello (3)</Text.Text>
            <Text.Text color={2}>Hello (2)</Text.Text>
            <Text.Text color={1}>Hello (1)</Text.Text>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Interactive Text (Links)"
        description="Text components with href attributes that render as clickable links"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Linked Text
          </Text.Text>
          <Flex.Box y>
            <Text.Text href="https://www.google.com" level="h1">
              Hello (H1 Link)
            </Text.Text>
            <Text.Text href="https://www.google.com" level="h2">
              Hello (H2 Link)
            </Text.Text>
            <Text.Text href="https://www.google.com" level="h3">
              Hello (H3 Link)
            </Text.Text>
            <Text.Text href="https://www.google.com" level="h4">
              Hello (H4 Link)
            </Text.Text>
            <Text.Text href="https://www.google.com" level="h5">
              Hello (H5 Link)
            </Text.Text>
            <Text.Text href="https://www.google.com">Hello (Body Link)</Text.Text>
            <Text.Text href="https://www.google.com" level="small">
              Hello (Small Link)
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Special Variants & Symbols"
        description="Code formatting, keyboard shortcuts, and special symbol components"
      >
        <Flex.Box y gap="medium">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Code Text
            </Text.Text>
            <Text.Text variant="code">Hello (Code)</Text.Text>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Keyboard Symbols
            </Text.Text>
            <Flex.Box x gap="small">
              <Text.Text variant="keyboard">
                <Icon.Keyboard.Control />
              </Text.Text>
              <Text.Text variant="keyboard">
                <Icon.Keyboard.Alt />
              </Text.Text>
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Trigger Display
            </Text.Text>
            <Triggers.Text level="h4" trigger={["Control", "Enter", "T"]} />
            <Triggers.Text level="h5" trigger={["Control", "Enter", "T"]} />
            <Triggers.Text level="p" trigger={["Control", "Enter", "T"]} />
            <Triggers.Text level="small" trigger={["Control", "Enter", "T"]} />
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Text Overflow Policies"
        description="How text behaves when it exceeds container width - ellipsis, clip, or nowrap"
      >
        <Flex.Box y gap="medium">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Ellipsis (default overflow with ...)
            </Text.Text>
            <Flex.Box
              style={{
                width: "200px",
                border: "1px solid var(--pluto-gray-l1)",
                padding: "8px",
              }}
            >
              <Text.Text overflow="ellipsis" style={{ width: "100px" }}>
                This is a very long text that will be truncated with ellipsis when it
                exceeds the container width
              </Text.Text>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Clip (hard cut without ...)
            </Text.Text>
            <Flex.Box
              style={{
                width: "200px",
                border: "1px solid var(--pluto-gray-l1)",
                padding: "8px",
              }}
            >
              <Text.Text overflow="clip">
                This is a very long text that will be clipped without any ellipsis
                indicator
              </Text.Text>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Nowrap (no wrapping, extends beyond container)
            </Text.Text>
            <Flex.Box
              style={{
                width: "200px",
                border: "1px solid var(--pluto-gray-l1)",
                padding: "8px",
                overflow: "visible",
              }}
            >
              <Text.Text overflow="nowrap">
                This text will not wrap and extends beyond the container boundaries
              </Text.Text>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Default (normal wrapping behavior)
            </Text.Text>
            <Flex.Box
              style={{
                width: "200px",
                border: "1px solid var(--pluto-gray-l1)",
                padding: "8px",
              }}
            >
              <Text.Text>
                This text will wrap normally when it reaches the container edge and
                continue on new lines
              </Text.Text>
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Status Variations"
        description="Text components with status variations"
      >
        <Flex.Box y gap="small">
          <Text.Text status="error">Error</Text.Text>
          <Text.Text status="warning">Warning</Text.Text>
          <Text.Text status="success">Success</Text.Text>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
