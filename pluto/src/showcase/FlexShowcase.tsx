// Copyright 2025 Synnax Labs, Inc.
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
import { InputShowcaseText } from "@/showcase/InputShowcase";
import { SubcategorySection } from "@/showcase/SubcategorySection";
import { Text } from "@/text";

const INPUT_PLACEHOLDER = (
  <>
    <Icon.Search />
    Catalyst
  </>
);

const DemoContainer = ({ children }: { children: React.ReactNode }) => (
  <Flex.Box
    style={{
      padding: "1.5rem",
      border: "1px solid var(--pluto-gray-l1)",
      borderRadius: "8px",
      background: "var(--pluto-gray-l0)",
    }}
  >
    {children}
  </Flex.Box>
);

export const FlexShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Basic Packed Layouts"
        description="Simple packed arrangements for 2-4 elements in horizontal and vertical directions"
      >
        <Flex.Box x gap="large" wrap>
          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Packed Single
            </Text.Text>
            <DemoContainer>
              <Flex.Box x pack>
                <Button.Button variant="filled">Hello</Button.Button>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Packed Pairs
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack>
                  <Button.Button variant="filled" rounded>
                    Hello
                  </Button.Button>
                  <Button.Button rounded>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box y pack rounded>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button variant="filled">Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Packed Triples
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button variant="filled">Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button variant="filled">Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Packed Quadruples
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box y pack>
                  <Flex.Box x pack>
                    <Button.Button variant="filled">Hello</Button.Button>
                    <Button.Button>Hello</Button.Button>
                  </Flex.Box>
                  <Flex.Box x pack>
                    <Button.Button>Hello</Button.Button>
                    <Button.Button variant="filled">Hello</Button.Button>
                  </Flex.Box>
                </Flex.Box>
                <Flex.Box x pack>
                  <Flex.Box y pack>
                    <Button.Button>Hello</Button.Button>
                    <Button.Button variant="filled">Hello</Button.Button>
                  </Flex.Box>
                  <Flex.Box y pack>
                    <Button.Button variant="filled">Hello</Button.Button>
                    <Button.Button>Hello</Button.Button>
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Grid Layouts"
        description="Complex grid arrangements using packed flex containers for uniform spacing"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              3x3 Grid (Rows)
            </Text.Text>
            <DemoContainer>
              <Flex.Box y pack>
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button variant="filled">Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              3x3 Grid (Columns)
            </Text.Text>
            <DemoContainer>
              <Flex.Box x pack>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button variant="filled">Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Nested Layouts"
        description="Complex nested arrangements for advanced UI patterns and groupings"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Nested Horizontal Packs
          </Text.Text>
          <DemoContainer>
            <Flex.Box y gap="small">
              <Flex.Box x pack>
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Button.Button variant="filled">Hello</Button.Button>
              </Flex.Box>
              <Flex.Box x pack>
                <Button.Button variant="filled">Hello</Button.Button>
                <Flex.Box x pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>
          </DemoContainer>

          <Text.Text level="small" weight={500}>
            Nested Vertical Packs
          </Text.Text>
          <DemoContainer>
            <Flex.Box x gap="small">
              <Flex.Box y pack>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
                <Button.Button variant="filled">Hello</Button.Button>
              </Flex.Box>
              <Flex.Box y pack>
                <Button.Button variant="filled">Hello</Button.Button>
                <Flex.Box y pack>
                  <Button.Button>Hello</Button.Button>
                  <Button.Button>Hello</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>
          </DemoContainer>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Form Input Patterns"
        description="Common UI patterns combining inputs with buttons for search and form interactions"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Search Input with Button
          </Text.Text>
          <DemoContainer>
            <Flex.Box pack>
              <InputShowcaseText placeholder={INPUT_PLACEHOLDER} />
              <Button.Button>
                <Icon.Search />
              </Button.Button>
            </Flex.Box>
          </DemoContainer>

          <Text.Text level="small" weight={500}>
            Stacked Search Forms
          </Text.Text>
          <DemoContainer>
            <Flex.Box x gap="small">
              <Flex.Box pack y>
                <InputShowcaseText placeholder="Search" />
                <Button.Button full="x" justify="center" variant="filled">
                  <Icon.Search />
                  Search
                </Button.Button>
              </Flex.Box>
              <Flex.Box pack y>
                <Button.Button full="x" justify="center" variant="filled">
                  <Icon.Search />
                  Search
                </Button.Button>
                <InputShowcaseText placeholder={INPUT_PLACEHOLDER} />
              </Flex.Box>
            </Flex.Box>
          </DemoContainer>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Non-Packed Layouts"
        description="Standard flexbox layouts without the pack optimization, using justify and align props"
      >
        <Flex.Box x gap="large" wrap>
          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Justify Center
            </Text.Text>
            <DemoContainer>
              <Flex.Box x justify="center" style={{ minHeight: "60px" }}>
                <Button.Button variant="filled">Hello</Button.Button>
                <Button.Button>World</Button.Button>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Justify Between
            </Text.Text>
            <DemoContainer>
              <Flex.Box x justify="between" style={{ minHeight: "60px" }}>
                <Button.Button variant="filled">Left</Button.Button>
                <Button.Button>Right</Button.Button>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Justify Around
            </Text.Text>
            <DemoContainer>
              <Flex.Box x justify="around" style={{ minHeight: "60px" }}>
                <Button.Button>A</Button.Button>
                <Button.Button variant="filled">B</Button.Button>
                <Button.Button>C</Button.Button>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Justify Evenly
            </Text.Text>
            <DemoContainer>
              <Flex.Box x justify="evenly" style={{ minHeight: "60px" }}>
                <Button.Button>A</Button.Button>
                <Button.Button variant="filled">B</Button.Button>
                <Button.Button>C</Button.Button>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Align Start/Center/End
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box
                  x
                  align="start"
                  style={{
                    minHeight: "50px",
                    border: "1px dashed var(--pluto-gray-l3)",
                  }}
                >
                  <Button.Button size="small">Start</Button.Button>
                  <Button.Button>Normal</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  align="center"
                  style={{
                    minHeight: "50px",
                    border: "1px dashed var(--pluto-gray-l3)",
                  }}
                >
                  <Button.Button size="small">Center</Button.Button>
                  <Button.Button>Normal</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  align="end"
                  style={{
                    minHeight: "50px",
                    border: "1px dashed var(--pluto-gray-l3)",
                  }}
                >
                  <Button.Button size="small">End</Button.Button>
                  <Button.Button>Normal</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Grow & Shrink
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x>
                  <Button.Button>Fixed</Button.Button>
                  <Button.Button grow>Grows to fill space</Button.Button>
                  <Button.Button>Fixed</Button.Button>
                </Flex.Box>
                <Flex.Box x>
                  <Button.Button grow={2}>Grow 2x</Button.Button>
                  <Button.Button grow>Grow 1x</Button.Button>
                  <Button.Button shrink>Shrink</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Styling Properties"
        description="Visual styling options including backgrounds, borders, colors, and corner styles"
      >
        <Flex.Box x gap="large" wrap>
          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Background Colors
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack background={1} style={{ padding: "0.5rem" }}>
                  <Button.Button>Background 1</Button.Button>
                  <Button.Button variant="filled">Light Gray</Button.Button>
                </Flex.Box>
                <Flex.Box x pack background={3} style={{ padding: "0.5rem" }}>
                  <Button.Button>Background 3</Button.Button>
                  <Button.Button variant="filled">Medium Gray</Button.Button>
                </Flex.Box>
                <Flex.Box x pack background={6} style={{ padding: "0.5rem" }}>
                  <Button.Button>Background 6</Button.Button>
                  <Button.Button variant="filled">Dark Gray</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Border Styles
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack bordered style={{ padding: "0.5rem" }}>
                  <Button.Button>Default Border</Button.Button>
                  <Button.Button variant="filled">Bordered</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  pack
                  bordered
                  borderColor={8}
                  borderWidth={2}
                  style={{ padding: "0.5rem" }}
                >
                  <Button.Button>Custom Border</Button.Button>
                  <Button.Button variant="filled">Thick & Dark</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  pack
                  bordered
                  borderColor="var(--pluto-primary-z)"
                  style={{ padding: "0.5rem" }}
                >
                  <Button.Button>Primary Border</Button.Button>
                  <Button.Button variant="filled">Colored</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Corner Styles
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack background={2} style={{ padding: "0.5rem" }}>
                  <Button.Button>Default Corners</Button.Button>
                  <Button.Button variant="filled">Normal</Button.Button>
                </Flex.Box>
                <Flex.Box x pack background={2} rounded style={{ padding: "0.5rem" }}>
                  <Button.Button>Rounded Corners</Button.Button>
                  <Button.Button variant="filled">Rounded</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  pack
                  background={2}
                  rounded={1.5}
                  style={{ padding: "0.5rem" }}
                >
                  <Button.Button>Custom Radius</Button.Button>
                  <Button.Button variant="filled">1.5rem</Button.Button>
                </Flex.Box>
                <Flex.Box x pack background={2} sharp style={{ padding: "0.5rem" }}>
                  <Button.Button>Sharp Corners</Button.Button>
                  <Button.Button variant="filled">No Radius</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Text Colors
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack color={7} background={1} style={{ padding: "0.5rem" }}>
                  <Text.Text>Dark text on light background</Text.Text>
                </Flex.Box>
                <Flex.Box x pack color={2} background={7} style={{ padding: "0.5rem" }}>
                  <Text.Text>Light text on dark background</Text.Text>
                </Flex.Box>
                <Flex.Box
                  x
                  pack
                  color="var(--pluto-primary-z)"
                  background={1}
                  style={{ padding: "0.5rem" }}
                >
                  <Text.Text>Primary colored text</Text.Text>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Size & Spacing
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box
                  x
                  pack
                  gap="small"
                  background={1}
                  style={{ padding: "0.5rem" }}
                >
                  <Button.Button size="small">Small Gap</Button.Button>
                  <Button.Button size="small">Between</Button.Button>
                  <Button.Button size="small">Items</Button.Button>
                </Flex.Box>
                <Flex.Box
                  x
                  pack
                  gap="large"
                  background={1}
                  style={{ padding: "0.5rem" }}
                >
                  <Button.Button>Large Gap</Button.Button>
                  <Button.Button>Between</Button.Button>
                  <Button.Button>Items</Button.Button>
                </Flex.Box>
                <Flex.Box x pack gap={2} background={1} style={{ padding: "0.5rem" }}>
                  <Button.Button>Custom Gap</Button.Button>
                  <Button.Button>2rem spacing</Button.Button>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>

          <Flex.Box y gap="medium">
            <Text.Text level="small" weight={500}>
              Full Width/Height & Square
            </Text.Text>
            <DemoContainer>
              <Flex.Box y gap="small">
                <Flex.Box x pack full="x" background={1} style={{ padding: "0.5rem" }}>
                  <Button.Button>Full Width</Button.Button>
                  <Button.Button variant="filled">Container</Button.Button>
                </Flex.Box>
                <Flex.Box x justify="center">
                  <Flex.Box square background={2} center size="medium">
                    <Icon.Calendar />
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            </DemoContainer>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>
  </Flex.Box>
);
