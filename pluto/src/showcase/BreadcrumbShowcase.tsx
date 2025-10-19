// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb } from "@/breadcrumb";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { SubcategorySection } from "@/showcase/SubcategorySection";
import { Text } from "@/text";

export const BreadcrumbShowcase = () => {
  const segments = [
    <Breadcrumb.Segment key="hardware">
      <Icon.Hardware />
      Hardware
    </Breadcrumb.Segment>,
    <Breadcrumb.Segment key="devices" href="https://www.google.com">
      Devices
    </Breadcrumb.Segment>,
    <Breadcrumb.Segment key="labjack" href="https://www.google.com">
      <Icon.Logo.LabJack />
      LabJack T7
    </Breadcrumb.Segment>,
  ];
  const URL = "https://docs.synnaxlabs.com/reference/core/cli-reference";
  return (
    <Flex.Box y pack empty>
      <SubcategorySection
        title="Typography Levels"
        description="Breadcrumbs with different text levels for various hierarchical contexts"
      >
        <Flex.Box y gap="medium">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              H4 Level
            </Text.Text>
            <Breadcrumb.Breadcrumb level="h4">{segments}</Breadcrumb.Breadcrumb>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              H5 Level
            </Text.Text>
            <Breadcrumb.Breadcrumb level="h5">{segments}</Breadcrumb.Breadcrumb>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Default Level
            </Text.Text>
            <Breadcrumb.Breadcrumb>{segments}</Breadcrumb.Breadcrumb>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Highlight Modes"
        description="Different highlighting styles to emphasize specific breadcrumb segments"
      >
        <Flex.Box y gap="medium">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Default (No Highlight)
            </Text.Text>
            <Breadcrumb.Breadcrumb color={9}>{segments}</Breadcrumb.Breadcrumb>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Highlight Last
            </Text.Text>
            <Breadcrumb.Breadcrumb highlightVariant="last">
              {segments}
            </Breadcrumb.Breadcrumb>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Highlight First
            </Text.Text>
            <Breadcrumb.Breadcrumb highlightVariant="first">
              {segments}
            </Breadcrumb.Breadcrumb>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Highlight All
            </Text.Text>
            <Breadcrumb.Breadcrumb highlightVariant="all">
              {segments}
            </Breadcrumb.Breadcrumb>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="URL-Based Breadcrumbs"
        description="Automatically generated breadcrumbs from URL paths using mapURLSegments utility"
      >
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Auto-generated from URL
          </Text.Text>
          <Breadcrumb.Breadcrumb>
            {Breadcrumb.mapURLSegments(URL, ({ href, segment }) => (
              <Breadcrumb.Segment key={segment} href={href}>
                {segment}
              </Breadcrumb.Segment>
            ))}
          </Breadcrumb.Breadcrumb>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  );
};
