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
import { Note } from "@/note";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

export const NoteShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Basic Note Variants"
        description="Different note variants for displaying information, warnings, and errors"
      >
        <Flex.Box y gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Info Note
            </Text.Text>
            <Note.Note variant="info">
              <Text.Text>
                <Icon.Info />
                This is an informational note with helpful details.
              </Text.Text>
            </Note.Note>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Warning Note
            </Text.Text>
            <Note.Note variant="warning">
              <Text.Text>
                <Icon.Warning />
                This is a warning note that alerts users to potential issues.
              </Text.Text>
            </Note.Note>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Error Note
            </Text.Text>
            <Note.Note variant="error">
              <Text.Text>
                <Icon.Close />
                This is an error note indicating something went wrong.
              </Text.Text>
            </Note.Note>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Note Content Variations"
        description="Notes with different content structures and layouts"
      >
        <Flex.Box y gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Text Only
            </Text.Text>
            <Note.Note variant="info">
              <Text.Text>Simple text-only note without icons.</Text.Text>
            </Note.Note>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Multi-line Content
            </Text.Text>
            <Note.Note variant="warning">
              <Flex.Box y gap="small">
                <Text.Text weight={500}>
                  <Icon.Warning />
                  Important Configuration Change
                </Text.Text>
                <Text.Text level="small" color={8} style={{ maxWidth: "300px" }}>
                  This action will modify your system settings. Please ensure you have
                  backed up your configuration before proceeding. This change cannot be
                  easily undone.
                </Text.Text>
              </Flex.Box>
            </Note.Note>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              With Action Elements
            </Text.Text>
            <Note.Note variant="error">
              <Flex.Box x gap="medium" align="center" justify="between">
                <Text.Text>
                  <Icon.Close />
                  Connection failed. Please check your network settings.
                </Text.Text>
                <Icon.Refresh style={{ cursor: "pointer", opacity: 0.7 }} />
              </Flex.Box>
            </Note.Note>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Note Sizing & Layout"
        description="Different note sizes and layout configurations"
      >
        <Flex.Box y gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Compact Notes
            </Text.Text>
            <Flex.Box y gap="tiny">
              <Note.Note variant="info">
                <Text.Text level="small">Compact info note</Text.Text>
              </Note.Note>
              <Note.Note variant="warning">
                <Text.Text level="small">Compact warning note</Text.Text>
              </Note.Note>
              <Note.Note variant="error">
                <Text.Text level="small">Compact error note</Text.Text>
              </Note.Note>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Inline Notes
            </Text.Text>
            <Flex.Box x gap="small" wrap>
              <Note.Note variant="info">
                <Text.Text level="small">Info</Text.Text>
              </Note.Note>
              <Note.Note variant="warning">
                <Text.Text level="small">Warning</Text.Text>
              </Note.Note>
              <Note.Note variant="error">
                <Text.Text level="small">Error</Text.Text>
              </Note.Note>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Full Width
            </Text.Text>
            <Note.Note variant="info" full="x">
              <Text.Text>This note spans the full width of its container.</Text.Text>
            </Note.Note>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Note Composition"
        description="Notes combined with other components and complex layouts"
      >
        <Flex.Box y gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Stacked Notes
            </Text.Text>
            <Flex.Box y gap="tiny">
              <Note.Note variant="info">
                <Text.Text level="small">Step 1: Configure your settings</Text.Text>
              </Note.Note>
              <Note.Note variant="warning">
                <Text.Text level="small">
                  Step 2: Review the configuration carefully
                </Text.Text>
              </Note.Note>
              <Note.Note variant="info">
                <Text.Text level="small">Step 3: Apply the changes</Text.Text>
              </Note.Note>
            </Flex.Box>
          </Flex.Box>

          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Nested Content
            </Text.Text>
            <Note.Note variant="warning">
              <Flex.Box y gap="small">
                <Text.Text weight={500}>System Requirements</Text.Text>
                <Flex.Box y gap="tiny" style={{ paddingLeft: "1rem" }}>
                  <Text.Text level="small">• Node.js 18 or higher</Text.Text>
                  <Text.Text level="small">• At least 4GB of RAM</Text.Text>
                  <Text.Text level="small">• 10GB of available disk space</Text.Text>
                </Flex.Box>
              </Flex.Box>
            </Note.Note>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
