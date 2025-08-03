// Copyright 2025 Synnax Labs, Inc.
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
import { InputShowcase } from "./InputShowcase";
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
  <Flex.Box
    y
    gap="large"
    style={{
      marginBottom: "4rem",
      padding: "2rem",
      background: "var(--pluto-gray-l0-5)",
      borderRadius: "1rem",
      border: "1px solid var(--pluto-gray-l1)",
      boxShadow: "0 1px 3px rgba(0, 0, 0, 0.05)",
    }}
  >
    <Flex.Box
      y
      gap="small"
      style={{ borderBottom: "1px solid var(--pluto-gray-l1)", paddingBottom: "1rem" }}
    >
      <Text.Text level="h2">{title}</Text.Text>
      {description && (
        <Text.Text
          level="p"
          style={{
            opacity: 0.8,
            maxWidth: "700px",
            lineHeight: 1.6,
            margin: 0,
          }}
        >
          {description}
        </Text.Text>
      )}
    </Flex.Box>
    <Flex.Box style={{ overflow: "auto" }}>{children}</Flex.Box>
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
      <Flex.Box
        y
        gap="medium"
        style={{
          position: "sticky",
          top: "1rem",
          zIndex: 10,
          background: "var(--pluto-gray-l0)",
          padding: "1.5rem",
          borderRadius: "0.75rem",
          border: "1px solid var(--pluto-gray-l2)",
          boxShadow: "0 2px 8px rgba(0, 0, 0, 0.1)",
        }}
      >
        <Text.Text level="h1" style={{ marginBottom: "0.5rem" }}>
          Pluto Component Showcase
        </Text.Text>
        <Text.Text level="p" style={{ opacity: 0.8, lineHeight: 1.5 }}>
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
