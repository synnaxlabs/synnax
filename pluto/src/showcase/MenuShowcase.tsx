// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

export const MenuShowcase = () => {
  const [basicSelected, setBasicSelected] = useState("option1");
  const [iconSelected, setIconSelected] = useState("save");
  const [iconOnlySelected, setIconOnlySelected] = useState("play");
  const [nestedSelected, setNestedSelected] = useState("file");

  const contextMenu = Menu.useContextMenu();

  const handleContextSelect = () => contextMenu.close();

  return (
    <Flex.Box y pack empty>
      <Flex.Box x pack grow sharp>
        <SubcategorySection
          title="Basic Menu"
          description="Simple menu with text items showing selection states and different sizes"
        >
          <Flex.Box x gap="large">
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Default Size
              </Text.Text>
              <Menu.Menu value={basicSelected} onChange={setBasicSelected}>
                <Menu.Item itemKey="option1">Option 1</Menu.Item>
                <Menu.Item itemKey="option2">Option 2</Menu.Item>
                <Menu.Item itemKey="option3">Option 3</Menu.Item>
                <Menu.Divider />
                <Menu.Item itemKey="option4">Option 4</Menu.Item>
              </Menu.Menu>
            </Flex.Box>
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Small Size
              </Text.Text>
              <Menu.Menu
                value={basicSelected}
                onChange={setBasicSelected}
                level="small"
              >
                <Menu.Item itemKey="option1" size="small">
                  Option 1
                </Menu.Item>
                <Menu.Item itemKey="option2" size="small">
                  Option 2
                </Menu.Item>
                <Menu.Item itemKey="option3" size="small">
                  Option 3
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item itemKey="option4" size="small">
                  Option 4
                </Menu.Item>
              </Menu.Menu>
            </Flex.Box>
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Large Size
              </Text.Text>
              <Menu.Menu value={basicSelected} onChange={setBasicSelected} level="h4">
                <Menu.Item itemKey="option1" size="large">
                  Option 1
                </Menu.Item>
                <Menu.Item itemKey="option2" size="large">
                  Option 2
                </Menu.Item>
                <Menu.Item itemKey="option3" size="large">
                  Option 3
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item itemKey="option4" size="large">
                  Option 4
                </Menu.Item>
              </Menu.Menu>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>

        <SubcategorySection
          title="Menu with Icons"
          description="Menu items with icons showing common UI patterns like file operations"
        >
          <Flex.Box
            y
            gap="small"
            style={{ maxWidth: "150px", padding: "1rem" }}
            bordered
            rounded
          >
            <Menu.Menu value={iconSelected} onChange={setIconSelected}>
              <Menu.Item itemKey="new">
                <Icon.Add />
                New File
              </Menu.Item>
              <Menu.Item itemKey="open">
                <Icon.Resources />
                Open
              </Menu.Item>
              <Menu.Item itemKey="save" trigger={["Control", "S"]} triggerIndicator>
                <Icon.Save />
                Save
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item itemKey="copy">
                <Icon.Copy />
                Copy
              </Menu.Item>
              <Menu.Item itemKey="edit">
                <Icon.Edit />
                Edit
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item itemKey="delete">
                <Icon.Delete />
                Delete
              </Menu.Item>
            </Menu.Menu>
          </Flex.Box>
        </SubcategorySection>

        <SubcategorySection
          title="Icon-Only Menu"
          description="Menu with only icons showing a compact toolbar-style interface"
        >
          <Flex.Box
            y
            gap="small"
            style={{ width: "fit-content", padding: "1rem" }}
            bordered
            rounded
          >
            <Menu.Menu value={iconOnlySelected} onChange={setIconOnlySelected}>
              <Menu.Item itemKey="play">
                <Icon.Play />
              </Menu.Item>
              <Menu.Item itemKey="pause">
                <Icon.Pause />
              </Menu.Item>
              <Menu.Item itemKey="stop">
                <Icon.Play />
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item itemKey="record">
                <Icon.Auto />
              </Menu.Item>
              <Menu.Item itemKey="rewind">
                <Icon.Refresh />
              </Menu.Item>
              <Menu.Item itemKey="fastforward">
                <Icon.Reference />
              </Menu.Item>
              <Menu.Divider />
              <Menu.Item itemKey="settings">
                <Icon.Settings />
              </Menu.Item>
            </Menu.Menu>
          </Flex.Box>
        </SubcategorySection>
      </Flex.Box>

      <Flex.Box x pack grow sharp>
        <SubcategorySection
          title="Background Contrast"
          description="Menu items on different background levels showing contrast adaptation"
        >
          <Flex.Box x gap="medium">
            <Flex.Box y background={1} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 1 Background
              </Text.Text>
              <Menu.Menu
                value={nestedSelected}
                onChange={setNestedSelected}
                background={1}
              >
                <Menu.Item itemKey="file">File</Menu.Item>
                <Menu.Item itemKey="edit">Edit</Menu.Item>
                <Menu.Item itemKey="view">View</Menu.Item>
              </Menu.Menu>
            </Flex.Box>
            <Flex.Box y background={2} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 2 Background
              </Text.Text>
              <Menu.Menu
                value={nestedSelected}
                onChange={setNestedSelected}
                background={2}
              >
                <Menu.Item itemKey="file">File</Menu.Item>
                <Menu.Item itemKey="edit">Edit</Menu.Item>
                <Menu.Item itemKey="view">View</Menu.Item>
              </Menu.Menu>
            </Flex.Box>
            <Flex.Box y background={3} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 3 Background
              </Text.Text>
              <Menu.Menu
                value={nestedSelected}
                onChange={setNestedSelected}
                background={3}
              >
                <Menu.Item itemKey="file">File</Menu.Item>
                <Menu.Item itemKey="edit">Edit</Menu.Item>
                <Menu.Item itemKey="view">View</Menu.Item>
              </Menu.Menu>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>

        <SubcategorySection
          title="Context Menu"
          description="Right-click menu that appears at cursor position with contextual actions"
        >
          <Flex.Box y gap="medium">
            <Text.Text level="small" color={8}>
              Right-click on the boxes below to see context menus
            </Text.Text>
            <Flex.Box x gap="medium">
              <Menu.ContextMenu
                {...contextMenu}
                menu={({ keys }) => (
                  <Menu.Menu onChange={handleContextSelect} level="small">
                    <Menu.Item itemKey="copy">
                      <Icon.Copy />
                      Copy ({keys.length} item{keys.length !== 1 ? "s" : ""})
                    </Menu.Item>
                    <Menu.Item itemKey="edit">
                      <Icon.Edit />
                      Edit
                    </Menu.Item>
                    <Menu.Item itemKey="refresh">
                      <Icon.Refresh />
                      Refresh
                    </Menu.Item>
                    <Menu.Divider />
                    <Menu.Item itemKey="rename">
                      <Icon.Rename />
                      Rename
                    </Menu.Item>
                    <Menu.Item itemKey="delete">
                      <Icon.Delete />
                      Delete
                    </Menu.Item>
                    <Menu.Divider />
                    <Menu.Item itemKey="properties">
                      <Icon.Settings />
                      Properties
                    </Menu.Item>
                  </Menu.Menu>
                )}
              >
                <Flex.Box x gap="medium">
                  <Flex.Box
                    className="pluto-context-target"
                    id="item-1"
                    background={1}
                    bordered
                    rounded={1}
                    onContextMenu={contextMenu.open}
                    style={{ padding: "2rem", cursor: "context-menu" }}
                  >
                    <Text.Text>Right-click me!</Text.Text>
                  </Flex.Box>
                  <Flex.Box
                    className="pluto-context-target"
                    id="item-2"
                    background={1}
                    bordered
                    rounded={1}
                    onContextMenu={contextMenu.open}
                    style={{ padding: "2rem", cursor: "context-menu" }}
                  >
                    <Text.Text>Or me!</Text.Text>
                  </Flex.Box>
                </Flex.Box>
              </Menu.ContextMenu>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>
      </Flex.Box>

      <Flex.Box x pack grow sharp>
        <SubcategorySection
          title="Menu Variations"
          description="Different menu configurations including compact spacing and custom triggers"
        >
          <Flex.Box x gap="large">
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                Compact Spacing
              </Text.Text>
              <Menu.Menu value={basicSelected} onChange={setBasicSelected} gap="tiny">
                <Menu.Item itemKey="option1" size="small">
                  Compact Item 1
                </Menu.Item>
                <Menu.Item itemKey="option2" size="small">
                  Compact Item 2
                </Menu.Item>
                <Menu.Item itemKey="option3" size="small">
                  Compact Item 3
                </Menu.Item>
              </Menu.Menu>
            </Flex.Box>
            <Flex.Box y gap="small">
              <Text.Text level="small" weight={500}>
                With Keyboard Shortcuts
              </Text.Text>
              <Menu.Menu value={iconSelected} onChange={setIconSelected}>
                <Menu.Item itemKey="new" trigger={["Control", "N"]}>
                  <Icon.Add />
                  New
                </Menu.Item>
                <Menu.Item itemKey="open" trigger={["Control", "O"]}>
                  <Icon.Resources />
                  Open
                </Menu.Item>
                <Menu.Item itemKey="save" trigger={["Control", "S"]}>
                  <Icon.Save />
                  Save
                </Menu.Item>
              </Menu.Menu>
            </Flex.Box>
          </Flex.Box>
        </SubcategorySection>
      </Flex.Box>
    </Flex.Box>
  );
};
