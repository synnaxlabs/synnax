// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { SubcategorySection } from "@/showcase/SubcategorySection";
import { Tag } from "@/tag";
import { Text } from "@/text";

export const TagShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Tag Sizes"
        description="Tags in different sizes from huge to tiny, showing consistent scaling"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Standard Tags
          </Text.Text>
          <Flex.Box y gap="medium">
            <Tag.Tag size="huge">Hello</Tag.Tag>
            <Tag.Tag size="large">Hello</Tag.Tag>
            <Tag.Tag size="medium">Hello</Tag.Tag>
            <Tag.Tag size="small">Hello</Tag.Tag>
            <Tag.Tag size="tiny">Hello</Tag.Tag>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Custom Colors"
        description="Tags with custom color overrides for branding or categorization"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Colored Tags
          </Text.Text>
          <Flex.Box y gap="medium">
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
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Closeable Tags"
        description="Tags with close buttons for removable labels and filters"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Removable Tags
          </Text.Text>
          <Flex.Box y gap="medium">
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
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
