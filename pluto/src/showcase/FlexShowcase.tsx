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
import { Text } from "@/text";

import { InputShowcaseText } from "./InputShowcase";
import { SubcategorySection } from "./SubcategorySection";

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
  </Flex.Box>
);
