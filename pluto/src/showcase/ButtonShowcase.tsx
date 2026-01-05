// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@/button";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

export const ButtonShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Sizes & Variants"
        description="Different button sizes (huge, large, medium, small, tiny) across all variants (default, filled, text, outlined)"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Default
            </Text.Text>
            <Button.Button size="huge">Hello</Button.Button>
            <Button.Button size="large">Hello</Button.Button>
            <Button.Button size="medium">Hello</Button.Button>
            <Button.Button size="small">Hello</Button.Button>
            <Button.Button size="tiny">Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Filled
            </Text.Text>
            <Button.Button size="huge" variant="filled">
              Hello
            </Button.Button>
            <Button.Button size="large" variant="filled">
              Hello
            </Button.Button>
            <Button.Button size="medium" variant="filled">
              Hello
            </Button.Button>
            <Button.Button size="small" variant="filled">
              Hello
            </Button.Button>
            <Button.Button size="tiny" variant="filled">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Text
            </Text.Text>
            <Button.Button size="huge" variant="text">
              Hello
            </Button.Button>
            <Button.Button size="large" variant="text">
              Hello
            </Button.Button>
            <Button.Button size="medium" variant="text">
              Hello
            </Button.Button>
            <Button.Button size="small" variant="text">
              Hello
            </Button.Button>
            <Button.Button size="tiny" variant="text">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Outlined
            </Text.Text>
            <Button.Button size="huge" variant="outlined">
              Hello
            </Button.Button>
            <Button.Button size="large" variant="outlined">
              Hello
            </Button.Button>
            <Button.Button size="medium" variant="outlined">
              Hello
            </Button.Button>
            <Button.Button size="small" variant="outlined">
              Hello
            </Button.Button>
            <Button.Button size="tiny" variant="outlined">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Shadow
            </Text.Text>
            <Button.Button size="huge" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button size="large" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button size="medium" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button size="small" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button size="tiny" variant="shadow">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Preview
            </Text.Text>
            <Button.Button size="huge" variant="preview">
              Hello
            </Button.Button>
            <Button.Button size="large" variant="preview">
              Hello
            </Button.Button>
            <Button.Button size="medium" variant="preview">
              Hello
            </Button.Button>
            <Button.Button size="small" variant="preview">
              Hello
            </Button.Button>
            <Button.Button size="tiny" variant="preview">
              Hello
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="With Icons"
        description="Buttons with text and icons, and icon-only buttons across different variants"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Text + Icon
            </Text.Text>
            <Button.Button size="large">
              <Icon.Add />
              Hello
            </Button.Button>
            <Button.Button size="large" variant="filled">
              <Icon.Add />
              Hello
            </Button.Button>
            <Button.Button size="large" variant="text">
              <Icon.Add />
              Hello
            </Button.Button>
            <Button.Button size="large" variant="outlined">
              <Icon.Add />
              Hello
            </Button.Button>
            <Button.Button size="large" variant="shadow">
              <Icon.Add />
              Hello
            </Button.Button>
            <Button.Button size="large" variant="preview">
              <Icon.Add />
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Icon Only
            </Text.Text>
            <Button.Button size="large">
              <Icon.Auto />
            </Button.Button>
            <Button.Button size="large" variant="filled">
              <Icon.Auto />
            </Button.Button>
            <Button.Button size="large" variant="text">
              <Icon.Auto />
            </Button.Button>
            <Button.Button size="large" variant="outlined">
              <Icon.Auto />
            </Button.Button>
            <Button.Button size="large" variant="shadow">
              <Icon.Auto />
            </Button.Button>
            <Button.Button size="large" variant="preview">
              <Icon.Auto />
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="States & Special Properties"
        description="Disabled, loading, sharp corners, and link buttons"
      >
        <Flex.Box y gap="medium">
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Disabled:
            </Text.Text>
            <Button.Button disabled>Hello</Button.Button>
            <Button.Button disabled variant="filled">
              Hello
            </Button.Button>
            <Button.Button disabled variant="text">
              Hello
            </Button.Button>
            <Button.Button disabled variant="shadow">
              Hello
            </Button.Button>
            <Button.Button disabled variant="preview">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Loading:
            </Text.Text>
            <Button.Button status="loading">Hello</Button.Button>
            <Button.Button status="loading">
              <Icon.Auto />
            </Button.Button>
          </Flex.Box>
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Sharp:
            </Text.Text>
            <Button.Button sharp>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Link:
            </Text.Text>
            <Button.Button href="https://www.google.com" variant="text">
              Link to Google
            </Button.Button>
            <Button.Button href="https://www.google.com" variant="filled">
              Link to Google
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Background Contrast"
        description="Buttons on different background colors to test contrast adaptation"
      >
        <Flex.Box x gap="medium">
          <Flex.Box y background={1} style={{ padding: "2rem" }} bordered rounded={1}>
            <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
              Level 1
            </Text.Text>
            <Button.Button contrast={1}>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y background={2} style={{ padding: "2rem" }} bordered rounded={1}>
            <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
              Level 2
            </Text.Text>
            <Button.Button contrast={2}>Hello</Button.Button>
          </Flex.Box>
          <Flex.Box y background={3} style={{ padding: "2rem" }} bordered rounded={1}>
            <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
              Level 3
            </Text.Text>
            <Button.Button contrast={3}>Hello</Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Status Colors"
        description="Buttons with warning and error status colors"
      >
        <Flex.Box y gap="medium">
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Warning:
            </Text.Text>
            <Button.Button status="warning">Hello</Button.Button>
            <Button.Button status="warning" variant="filled">
              Hello
            </Button.Button>
            <Button.Button status="warning" variant="text">
              Hello
            </Button.Button>
            <Button.Button status="warning" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button status="warning" variant="preview">
              Hello
            </Button.Button>
          </Flex.Box>
          <Flex.Box x gap="medium">
            <Text.Text level="small" weight={500}>
              Error:
            </Text.Text>
            <Button.Button status="error">Hello</Button.Button>
            <Button.Button status="error" variant="filled">
              Hello
            </Button.Button>
            <Button.Button status="error" variant="text">
              Hello
            </Button.Button>
            <Button.Button status="error" variant="shadow">
              Hello
            </Button.Button>
            <Button.Button status="error" variant="preview">
              Hello
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Custom Colors"
        description="Buttons with custom color overrides"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Custom Color Examples
          </Text.Text>
          <Flex.Box x gap="medium" wrap>
            <Button.Button color="#12E3E2">Hello</Button.Button>
            <Button.Button color="#12E3E2" variant="filled">
              Hello
            </Button.Button>
            <Button.Button color="#12E3E2" variant="text">
              Hello
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Trigger Indicators"
        description="Buttons with keyboard shortcuts displayed using showTriggerIndicator"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Buttons with Trigger Indicators
          </Text.Text>
          <Flex.Box x gap="medium" wrap>
            <Button.Button trigger={["Control", "S"]} triggerIndicator>
              Save
            </Button.Button>
            <Button.Button variant="filled" trigger={["Control", "N"]} triggerIndicator>
              <Icon.Add />
              New
            </Button.Button>
            <Button.Button variant="text" trigger={["Control", "D"]} triggerIndicator>
              Delete
            </Button.Button>
            <Button.Button
              variant="outlined"
              trigger={["Control", "Shift", "Z"]}
              triggerIndicator
            >
              <Icon.Release />
              Redo
            </Button.Button>
            <Button.Button size="small" trigger={["Escape"]} triggerIndicator>
              Cancel
            </Button.Button>
            <Button.Button trigger={["Enter"]} triggerIndicator>
              Confirm
            </Button.Button>
            <Button.Button variant="text" trigger={["T"]} triggerIndicator size="tiny">
              <Icon.Auto />
            </Button.Button>
          </Flex.Box>
          <Text.Text level="small" weight={500}>
            Large & Huge Sizes with Triggers
          </Text.Text>
          <Flex.Box x gap="medium" wrap>
            <Button.Button size="large" trigger={["Control", "O"]} triggerIndicator>
              <Icon.Attachment />
              Open File
            </Button.Button>
            <Button.Button
              size="large"
              variant="filled"
              trigger={["Control", "Shift", "S"]}
              triggerIndicator
            >
              Save As
            </Button.Button>
            <Button.Button size="huge" trigger={["F5"]} triggerIndicator>
              <Icon.Refresh />
              Refresh
            </Button.Button>
            <Button.Button
              size="huge"
              variant="outlined"
              trigger={["Control", "Z"]}
              triggerIndicator
            >
              Undo
            </Button.Button>
            <Button.Button
              size="large"
              variant="text"
              trigger={["Control", "Q"]}
              triggerIndicator
            >
              Quit
            </Button.Button>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
