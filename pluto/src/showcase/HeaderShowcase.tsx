// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";

import { SubcategorySection } from "./SubcategorySection";

export const HeaderShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Basic headers"
        description="Headers with different text levels and basic structure"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h1">
            <Header.Title>Large header (h1)</Header.Title>
          </Header.Header>
          <Header.Header level="h2">
            <Header.Title>Medium header (h2)</Header.Title>
          </Header.Header>
          <Header.Header level="h3">
            <Header.Title>Small header (h3)</Header.Title>
          </Header.Header>
          <Header.Header level="p">
            <Header.Title>Paragraph level header</Header.Title>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Headers with icons"
        description="Headers with icons in the title"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h2">
            <Header.Title>
              <Icon.Settings />
              Settings Header
            </Header.Title>
          </Header.Header>
          <Header.Header level="h2">
            <Header.Title>
              <Icon.Device />
              Database Configuration
            </Header.Title>
          </Header.Header>
          <Header.Header level="h3">
            <Header.Title>
              <Icon.User />
              User Profile
            </Header.Title>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Headers with actions"
        description="Headers with action buttons on the right side"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h2">
            <Header.Title>Project settings</Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Edit />
              </Button.Button>
              <Button.Button variant="text">
                <Icon.Close />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header level="h3">
            <Header.Title>
              <Icon.Resources />
              File Manager
            </Header.Title>
            <Header.Actions>
              <Button.Button variant="filled" size="small">
                <Icon.Add />
                New File
              </Button.Button>
              <Button.Button variant="outlined" size="small">
                <Icon.Download />
                Upload
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header>
            <Header.Title>Dashboard</Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Refresh />
              </Button.Button>
              <Button.Button variant="text">
                <Icon.Settings />
              </Button.Button>
              <Button.Button variant="text">
                <Icon.Release />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Clickable headers"
        description="Headers with clickable titles using ButtonTitle"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h2">
            <Header.ButtonTitle onClick={() => console.log("Header clicked")}>
              <Icon.Caret.Right />
              Expandable Section
            </Header.ButtonTitle>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Menu />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header level="h3">
            <Header.ButtonTitle onClick={() => console.log("Navigation clicked")}>
              <Icon.Arrow.Left />
              Back to Home
            </Header.ButtonTitle>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Divided headers"
        description="Headers with dividers between elements"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h2" divided>
            <Header.Title>
              <Icon.Device />
              Data Configuration
            </Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Save />
              </Button.Button>
              <Button.Button variant="text">
                <Icon.Close />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header level="h3" divided>
            <Header.ButtonTitle onClick={() => console.log("Divided header clicked")}>
              <Icon.Resources />
              Project Folder
            </Header.ButtonTitle>
            <Header.Actions>
              <Button.Button variant="filled" size="small">
                Open
              </Button.Button>
            </Header.Actions>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Border variations"
        description="Headers with and without borders"
      >
        <Flex.Box y gap="large">
          <Text.Text level="small" weight={500}>
            With Border (default)
          </Text.Text>
          <Header.Header level="h3" bordered>
            <Header.Title>Bordered header</Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Settings />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Text.Text level="small" weight={500}>
            Without Border
          </Text.Text>
          <Header.Header level="h3" bordered={false}>
            <Header.Title>Borderless header</Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Settings />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Complex headers"
        description="Headers with multiple actions and complex layouts"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h2" divided>
            <Header.Title>
              <Icon.Core />
              Cluster Management
            </Header.Title>
            <Header.Actions>
              <Button.Button variant="text" status="warning">
                <Icon.Warning />
                Issues
              </Button.Button>
              <Button.Button variant="outlined" size="small">
                <Icon.Refresh />
                Refresh
              </Button.Button>
              <Button.Button variant="filled" size="small">
                <Icon.Add />
                Add Node
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header level="h3">
            <Header.ButtonTitle onClick={() => console.log("Device clicked")}>
              <Icon.Hardware />
              Device Monitor
              <Icon.Link />
            </Header.ButtonTitle>
            <Header.Actions>
              <Button.Button variant="text" size="small" status="error">
                <Icon.Release />
                Stop
              </Button.Button>
              <Button.Button variant="text" size="small">
                <Icon.Play />
                Start
              </Button.Button>
              <Button.Button variant="text" size="small">
                <Icon.Release />
                Restart
              </Button.Button>
              <Button.Button variant="text" size="small">
                <Icon.Release />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Custom styling"
        description="Headers with custom spacing and background variations"
      >
        <Flex.Box y gap="large">
          <Header.Header level="h3" style={{ padding: "2rem" }} background={1}>
            <Header.Title>Header with custom padding</Header.Title>
            <Header.Actions>
              <Button.Button variant="text">
                <Icon.Settings />
              </Button.Button>
            </Header.Actions>
          </Header.Header>
          <Header.Header level="h3" background={2} style={{ padding: "1.5rem" }}>
            <Header.Title>
              <Icon.Delete />
              Theme Settings
            </Header.Title>
            <Header.Actions>
              <Button.Button variant="filled" size="small">
                Apply
              </Button.Button>
            </Header.Actions>
          </Header.Header>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
