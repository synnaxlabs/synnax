// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Button } from "@/button";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Nav } from "@/nav";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

export const NavShowcase = () => {
  const [topDrawerItems] = useState([
    {
      key: "files",
      content: (
        <Flex.Box y gap="medium" style={{ padding: "2rem" }}>
          <Text.Text level="h4">Files</Text.Text>
          <Flex.Box y gap="small">
            <Text.Text>document.txt</Text.Text>
            <Text.Text>image.png</Text.Text>
            <Text.Text>config.json</Text.Text>
          </Flex.Box>
        </Flex.Box>
      ),
      initialSize: 200,
      minSize: 150,
      maxSize: 400,
    },
    {
      key: "search",
      content: (
        <Flex.Box y gap="medium" style={{ padding: "2rem" }}>
          <Text.Text level="h4">Search Results</Text.Text>
          <Text.Text level="small" color={8}>
            3 matches found
          </Text.Text>
        </Flex.Box>
      ),
      initialSize: 180,
      minSize: 120,
      maxSize: 350,
    },
  ]);

  const [leftDrawerItems] = useState([
    {
      key: "explorer",
      content: (
        <Flex.Box y gap="medium" style={{ padding: "2rem" }}>
          <Text.Text level="h4">Explorer</Text.Text>
          <Flex.Box y gap="small">
            <Text.Text>📁 src/</Text.Text>
            <Text.Text>📁 docs/</Text.Text>
            <Text.Text>📄 README.md</Text.Text>
          </Flex.Box>
        </Flex.Box>
      ),
      initialSize: 250,
      minSize: 200,
      maxSize: 400,
    },
    {
      key: "outline",
      content: (
        <Flex.Box y gap="medium" style={{ padding: "2rem" }}>
          <Text.Text level="h4">Outline</Text.Text>
          <Flex.Box y gap="small">
            <Text.Text>1. Introduction</Text.Text>
            <Text.Text>2. Getting Started</Text.Text>
            <Text.Text>3. API Reference</Text.Text>
          </Flex.Box>
        </Flex.Box>
      ),
      initialSize: 220,
      minSize: 180,
      maxSize: 350,
    },
  ]);

  const topDrawer = Nav.useDrawer({ items: topDrawerItems });
  const leftDrawer = Nav.useDrawer({ items: leftDrawerItems });

  return (
    <Flex.Box y pack empty>
      <Flex.Box x pack grow sharp>
        <SubcategorySection
          title="Navigation Bar"
          description="Horizontal and vertical navigation bars with different locations and content alignment"
        >
          <Flex.Box y gap="large">
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Top Navigation Bar
              </Text.Text>
              <Flex.Box
                style={{
                  height: "200px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Nav.Bar location="top" size="4rem" bordered>
                  <Nav.Bar.Start>
                    <Button.Button>
                      <Icon.Menu />
                    </Button.Button>
                    <Text.Text level="h4">App Title</Text.Text>
                  </Nav.Bar.Start>
                  <Nav.Bar.Center>
                    <Button.Button size="small">Home</Button.Button>
                    <Button.Button size="small">About</Button.Button>
                    <Button.Button size="small">Contact</Button.Button>
                  </Nav.Bar.Center>
                  <Nav.Bar.End>
                    <Button.Button>
                      <Icon.Settings />
                    </Button.Button>
                    <Button.Button>
                      <Icon.User />
                    </Button.Button>
                  </Nav.Bar.End>
                </Nav.Bar>
                <Flex.Box
                  style={{
                    padding: "2rem",
                    height: "calc(100% - 4rem)",
                    marginTop: "4rem",
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>Main content area</Text.Text>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>

            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Left Navigation Bar
              </Text.Text>
              <Flex.Box
                style={{
                  height: "200px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Nav.Bar location="left" size="6rem" bordered>
                  <Nav.Bar.Start>
                    <Button.Button>
                      <Icon.Resources />
                    </Button.Button>
                  </Nav.Bar.Start>
                  <Nav.Bar.Center>
                    <Button.Button>
                      <Icon.Resources />
                    </Button.Button>
                    <Button.Button>
                      <Icon.Search />
                    </Button.Button>
                    <Button.Button>
                      <Icon.Settings />
                    </Button.Button>
                  </Nav.Bar.Center>
                  <Nav.Bar.End>
                    <Button.Button>
                      <Icon.User />
                    </Button.Button>
                  </Nav.Bar.End>
                </Nav.Bar>
                <Flex.Box
                  style={{
                    padding: "2rem",
                    width: "calc(100% - 3rem)",
                    marginLeft: "6rem",
                    height: "100%",
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>Main content area</Text.Text>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>

        <SubcategorySection
          title="Navigation Bar Variations"
          description="Different navigation bar sizes and content arrangements"
        >
          <Flex.Box y gap="large">
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Bottom Navigation Bar
              </Text.Text>
              <Flex.Box
                style={{
                  height: "150px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Flex.Box
                  style={{
                    padding: "2rem",
                    height: "calc(100% - 3rem)",
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>Main content area</Text.Text>
                </Flex.Box>
                <Nav.Bar location="bottom" size="6rem" bordered>
                  <Nav.Bar.Center>
                    <Button.Button size="small" variant="outlined">
                      <Icon.Resources />
                      Home
                    </Button.Button>
                    <Button.Button size="small" variant="outlined">
                      <Icon.Resources />
                      Files
                    </Button.Button>
                    <Button.Button size="small" variant="outlined">
                      <Icon.Settings />
                      Settings
                    </Button.Button>
                  </Nav.Bar.Center>
                </Nav.Bar>
              </Flex.Box>
            </Flex.Box>

            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Right Navigation Bar
              </Text.Text>
              <Flex.Box
                style={{
                  height: "150px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Flex.Box
                  style={{
                    padding: "2rem",
                    width: "calc(100% - 4rem)",
                    height: "100%",
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>Main content area</Text.Text>
                </Flex.Box>
                <Nav.Bar location="right" size="4rem" bordered>
                  <Nav.Bar.Start>
                    <Button.Button>
                      <Icon.Add />
                    </Button.Button>
                  </Nav.Bar.Start>
                  <Nav.Bar.Center>
                    <Button.Button>
                      <Icon.Save />
                    </Button.Button>
                    <Button.Button>
                      <Icon.Copy />
                    </Button.Button>
                    <Button.Button>
                      <Icon.Edit />
                    </Button.Button>
                  </Nav.Bar.Center>
                  <Nav.Bar.End>
                    <Button.Button>
                      <Icon.Delete />
                    </Button.Button>
                  </Nav.Bar.End>
                </Nav.Bar>
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>
      </Flex.Box>

      <Flex.Box x pack grow sharp>
        <SubcategorySection
          title="Navigation Drawer"
          description="Collapsible drawer panels that can be toggled and resized for additional navigation content"
        >
          <Flex.Box y gap="large">
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Top Drawer with Controls
              </Text.Text>
              <Flex.Box
                style={{
                  height: "300px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Nav.Bar location="top" size="6rem" bordered>
                  <Nav.Bar.Start>
                    <Button.Button
                      size="small"
                      variant={
                        topDrawer.activeItem?.key === "files" ? "filled" : "text"
                      }
                      onClick={() => topDrawer.onSelect?.("files")}
                    >
                      <Icon.Resources />
                      Files
                    </Button.Button>
                    <Button.Button
                      size="small"
                      variant={
                        topDrawer.activeItem?.key === "search" ? "filled" : "text"
                      }
                      onClick={() => topDrawer.onSelect?.("search")}
                    >
                      <Icon.Search />
                      Search
                    </Button.Button>
                  </Nav.Bar.Start>
                  <Nav.Bar.End>
                    <Button.Button>
                      <Icon.Settings />
                    </Button.Button>
                  </Nav.Bar.End>
                </Nav.Bar>
                <Nav.Drawer {...topDrawer} location="top" />
                <Flex.Box
                  style={{
                    padding: "2rem",
                    height: `calc(100% - 3rem - ${
                      topDrawer.activeItem
                        ? `${topDrawer.activeItem.initialSize}px`
                        : "0px"
                    })`,
                    marginTop: `calc(3rem + ${
                      topDrawer.activeItem
                        ? `${topDrawer.activeItem.initialSize}px`
                        : "0px"
                    })`,
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>
                    {topDrawer.activeItem
                      ? `${topDrawer.activeItem.key} drawer is open`
                      : "Click buttons to open drawers"}
                  </Text.Text>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>

            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Left Drawer with Controls
              </Text.Text>
              <Flex.Box
                style={{
                  height: "300px",
                  border: "1px solid var(--pluto-border-color)",
                  position: "relative",
                }}
              >
                <Nav.Bar location="left" size="6rem" bordered>
                  <Nav.Bar.Start>
                    <Button.Button
                      variant={
                        leftDrawer.activeItem?.key === "explorer" ? "filled" : "text"
                      }
                      onClick={() => leftDrawer.onSelect?.("explorer")}
                    >
                      <Icon.Resources />
                    </Button.Button>
                    <Button.Button
                      variant={
                        leftDrawer.activeItem?.key === "outline" ? "filled" : "text"
                      }
                      onClick={() => leftDrawer.onSelect?.("outline")}
                    >
                      <Icon.Menu />
                    </Button.Button>
                  </Nav.Bar.Start>
                  <Nav.Bar.End>
                    <Button.Button>
                      <Icon.Settings />
                    </Button.Button>
                  </Nav.Bar.End>
                </Nav.Bar>
                <Nav.Drawer {...leftDrawer} location="left" />
                <Flex.Box
                  style={{
                    padding: "2rem",
                    width: `calc(100% - 3rem - ${
                      leftDrawer.activeItem
                        ? `${leftDrawer.activeItem.initialSize}px`
                        : "0px"
                    })`,
                    marginLeft: `calc(3rem + ${
                      leftDrawer.activeItem
                        ? `${leftDrawer.activeItem.initialSize}px`
                        : "0px"
                    })`,
                    height: "100%",
                  }}
                  align="center"
                  justify="center"
                >
                  <Text.Text color={8}>
                    {leftDrawer.activeItem
                      ? `${leftDrawer.activeItem.key} drawer is open`
                      : "Click buttons to open drawers"}
                  </Text.Text>
                </Flex.Box>
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>
      </Flex.Box>
    </Flex.Box>
  );
};
