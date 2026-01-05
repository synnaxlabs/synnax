// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { state } from "@/state";
import { Text } from "@/text";

import { BreadcrumbShowcase } from "./BreadcrumbShowcase";
import { ButtonShowcase } from "./ButtonShowcase";
import { DISPLAY, PADDING_STYLE } from "./constants";
import { DisplaySelector } from "./DisplaySelector";
import { DividerShowcase } from "./DividerShowcase";
import { FlexShowcase } from "./FlexShowcase";
import { HeaderShowcase } from "./HeaderShowcase";
import { InputShowcase } from "./InputShowcase";
import { MenuShowcase } from "./MenuShowcase";
import { NoteShowcase } from "./NoteShowcase";
import { SelectShowcase } from "./SelectShowcase";
import { TagShowcase } from "./TagShowcase";
import { TextShowcase } from "./TextShowcase";

const ShowcaseSection = ({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) => (
  <Flex.Box y gap="large" background={1} rounded={2} pack>
    <Flex.Box y gap="small" bordered style={{ padding: "2rem 3rem" }} rounded>
      <Text.Text level="h3">{title}</Text.Text>
      {description && (
        <Text.Text level="p" color={8}>
          {description}
        </Text.Text>
      )}
    </Flex.Box>
    {children}
  </Flex.Box>
);

export const Showcase = () => {
  const [display, setDisplay] = state.usePersisted<(typeof DISPLAY)[number][]>(
    DISPLAY,
    "display",
  );

  const showcaseItems = [
    {
      key: "text",
      title: "Text",
      description:
        "Typography components for headings, body text, and inline elements with consistent styling and hierarchy.",
      component: <TextShowcase />,
    },
    {
      key: "button",
      title: "Button",
      description:
        "Interactive button components with various sizes, variants, states, and styling options.",
      component: <ButtonShowcase />,
    },
    {
      key: "input",
      title: "Input",
      description:
        "Form input components including text fields, numbers, dates, and switches with validation support.",
      component: <InputShowcase />,
    },
    {
      key: "select",
      title: "Select",
      description:
        "Dropdown selection components for single and multiple value selection with search capabilities.",
      component: <SelectShowcase />,
    },
    {
      key: "menu",
      title: "Menu",
      description:
        "Interactive menu components with selection states, icons, and context menu functionality.",
      component: <MenuShowcase />,
    },
    {
      key: "note",
      title: "Note",
      description:
        "Alert and notification components for displaying informational, warning, and error messages.",
      component: <NoteShowcase />,
    },
    {
      key: "tag",
      title: "Tag",
      description:
        "Label and tag components for categorization, status indication, and metadata display.",
      component: <TagShowcase />,
    },
    {
      key: "flex",
      title: "Flex",
      description:
        "Layout components using flexbox for responsive and flexible content arrangement.",
      component: <FlexShowcase />,
    },
    {
      key: "header",
      title: "Header",
      description:
        "Header components with titles, actions, and dividers for module and section organization.",
      component: <HeaderShowcase />,
    },
    {
      key: "breadcrumb",
      title: "Breadcrumb",
      description:
        "Navigation breadcrumb components for hierarchical path display and navigation.",
      component: <BreadcrumbShowcase />,
    },
    {
      key: "divider",
      title: "Divider",
      description:
        "Visual separator components for content organization and layout structure.",
      component: <DividerShowcase />,
    },
  ];

  return (
    <Flex.Box
      y
      gap="large"
      style={{
        ...PADDING_STYLE,
        maxWidth: "1400px",
        margin: "0 auto",
        padding: "min(5rem, 5vw)",
      }}
    >
      <Flex.Box y background={1} bordered rounded={2} style={{ padding: "5rem" }}>
        <Text.Text level="h2" style={{ marginBottom: "0.5rem" }}>
          Pluto Component Showcase
        </Text.Text>
        <Text.Text level="p" color={9}>
          Interactive showcase of all Pluto design system components. Select components
          to view their variants and usage examples.
        </Text.Text>
        <DisplaySelector display={display} setDisplay={setDisplay} />
      </Flex.Box>

      <Flex.Box y gap="huge" style={{ minHeight: "50vh" }}>
        {showcaseItems.length > 0 && display.length === 0 && (
          <Flex.Box
            y
            gap="medium"
            align="center"
            justify="center"
            style={{
              padding: "4rem 2rem",
              textAlign: "center",
              opacity: 0.6,
            }}
          >
            <Text.Text level="h3">No Components Selected</Text.Text>
            <Text.Text level="p">
              Choose components from the selector above to view their showcases.
            </Text.Text>
          </Flex.Box>
        )}

        {showcaseItems.map(
          ({ key, title, description, component }) =>
            display.includes(key) && (
              <ShowcaseSection key={key} title={title} description={description}>
                {component}
              </ShowcaseSection>
            ),
        )}
      </Flex.Box>
    </Flex.Box>
  );
};
