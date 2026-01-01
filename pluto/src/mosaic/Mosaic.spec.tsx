// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Mosaic } from "@/mosaic";

const TestMosaic = (props: Mosaic.UseProps): ReactElement => {
  const props_ = Mosaic.use(props);
  return <Mosaic.Mosaic {...props_}>{() => null}</Mosaic.Mosaic>;
};

describe("Mosaic", () => {
  describe("mosaicTree", () => {
    describe("Mosaic.insertTab", () => {
      it("should insert a tab into the center of an empty tree", () => {
        const tab = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tree = Mosaic.insertTab({ key: 1, tabs: [] }, tab, "center", 1);
        expect(tree).toEqual({
          key: 1,
          tabs: [tab],
          selected: "tab1",
        });
      });
      it("shouldn't split an empty tree with one tab, instead put the tab in the center", () => {
        const tab = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tree = Mosaic.insertTab({ key: 1, tabs: [] }, tab, "right", 1);
        expect(tree).toEqual({
          key: 1,
          tabs: [tab],
          selected: "tab1",
        });
      });
      it("should split a tree with one tab", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };

        const tree = {
          key: 1,
          tabs: [tabOne],
          selected: "tab1",
        };
        const nextTree = Mosaic.insertTab(tree, tabTwo, "right", 1);

        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        });
      });
      it("should insert a tab into the center of a valid leaf when no key or location is provided", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        };
        const nextTree = Mosaic.insertTab(tree, tabThree);
        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne, tabThree],
            selected: "tab3",
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        });
      });
    });

    describe("Mosaic.removeTab", () => {
      it("should remove a tab from the center of a tree", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
            selected: "tab2",
          },
        };
        const [nextTree] = Mosaic.removeTab(tree, "tab2");
        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabThree],
            selected: "tab3",
          },
        });
      });
      it("should remove a tab from the the side of a tree and garbage collect", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
            selected: "tab2",
          },
        };
        const [nextTree] = Mosaic.removeTab(tree, "tab1");
        expect(nextTree).toEqual({
          key: 1,
          tabs: [tabTwo, tabThree],
          selected: "tab2",
        });
      });
    });

    describe("Mosaic.moveTab", () => {
      it("should move a tab from one side of a leaf to another", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
            selected: "tab2",
          },
        };
        const [nextTree] = Mosaic.moveTab(tree, "tab2", "center", 2);
        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne, tabTwo],
            selected: "tab2",
          },
          last: {
            key: 3,
            tabs: [tabThree],
            selected: "tab3",
          },
        });
      });
      it("should correctly move a tab from the first leaf  of the root Mosaic.Node to the second leaf", () => {
        const tabOne = {
          tabKey: "1",
          name: "Tab 1",
          content: "Tab One Content",
        };
        const tabTwo = {
          tabKey: "2",
          name: "Tab 2",
          content: "Tab Two Content",
        };
        const tabThree = {
          tabKey: "3",
          name: "Tab 3",
          content: "Tab Three Content",
        };

        const initialTree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
          },
        };
        const [nextTree] = Mosaic.moveTab(initialTree, "1", "center", 2);
        expect(nextTree).toEqual({
          key: 1,
          selected: "1",
          size: undefined,
          tabs: [tabTwo, tabThree, tabOne],
        });
      });
      it("should maintain correct key hierarchy after moving tabs and garbage collection", () => {
        // Initial state matching the bug scenario
        const initialTree: Mosaic.Node = {
          key: 1,
          direction: "y",
          last: {
            key: 3,
            tabs: [
              {
                closable: true,
                icon: "Logo.LabJack",
                name: "LabJack Read Task",
                tabKey: "a35c8a98-7a37-4365-a0b4-0fd38cedfdb8",
              },
            ],
            selected: "a35c8a98-7a37-4365-a0b4-0fd38cedfdb8",
          },
          first: {
            key: 2,
            direction: "x",
            last: {
              key: 5,
              tabs: [
                {
                  closable: true,
                  icon: "Visualize",
                  name: "New Component",
                  tabKey: "ff989ff2-9bdb-49fe-b6fc-d71c8f933309",
                },
              ],
              selected: "ff989ff2-9bdb-49fe-b6fc-d71c8f933309",
              size: 0.5,
            },
            first: {
              key: 4,
              tabs: [
                {
                  closable: true,
                  icon: "Visualize",
                  name: "New Component",
                  tabKey: "dbca1c5e-7d69-4ac6-bb59-22c9d2bf3ee3",
                },
              ],
              selected: "dbca1c5e-7d69-4ac6-bb59-22c9d2bf3ee3",
            },
          },
        };

        // Move the LabJack tab
        const [result] = Mosaic.moveTab(
          initialTree,
          "a35c8a98-7a37-4365-a0b4-0fd38cedfdb8",
          "bottom",
          5,
        );

        // Helper function to verify key hierarchy
        const verifyKeyHierarchy = (node: Mosaic.Node): boolean => {
          let valid = true;
          if (node.first) {
            valid &&= node.first.key === node.key * 2;
            valid &&= verifyKeyHierarchy(node.first);
          }
          if (node.last) {
            valid &&= node.last.key === node.key * 2 + 1;
            valid &&= verifyKeyHierarchy(node.last);
          }
          return valid;
        };

        expect(result.key).toBe(1);

        expect(verifyKeyHierarchy(result)).toBe(true);
      });
    });

    describe("Mosaic.resizeLeaf", () => {
      it("should resize a leaf", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        };
        const nextTree = Mosaic.resizeNode(tree, 2, 100);
        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
            size: 100,
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        });
      });
    });

    describe("Mosaic.selectTab", () => {
      it("should select a tab", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab1",
        };
        const nextTree = Mosaic.selectTab(tree, "tab2");
        expect(nextTree).toEqual({
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab2",
        });
      });
    });

    describe("Mosaic.renameTab", () => {
      it("should rename a tab", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab1",
        };
        const nextTree = Mosaic.renameTab(tree, "tab1", "New Tab 1");
        expect(nextTree).toEqual({
          key: 1,
          tabs: [{ tabKey: "tab1", name: "New Tab 1" }, tabTwo],
          selected: "tab1",
        });
      });
    });

    describe("Mosaic.findTabNode", () => {
      it("should find a tab", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab1",
        };
        const node = Mosaic.findTabNode(tree, "tab2");
        expect(node?.key).toEqual(1);
      });
      it("should find a tab in a nested tree", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
            selected: "tab2",
          },
        };
        const node = Mosaic.findTabNode(tree, "tab3");
        expect(node?.key).toEqual(3);
      });
      it("should return undefined if the tab is not found", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab1",
        };
        const node = Mosaic.findTabNode(tree, "tab3");
        expect(node).toBeUndefined();
      });
    });

    describe("Mosaic.splitVertically", () => {
      it("should split a tree vertically", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tree: Mosaic.Node = {
          key: 1,
          tabs: [tabOne, tabTwo],
          selected: "tab1",
        };
        const nextTree = Mosaic.split(tree, tabTwo.tabKey, "y");
        expect(nextTree).toEqual({
          key: 1,
          direction: "y",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            tabs: [tabTwo],
            selected: "tab2",
          },
        });
      });

      it("should split a nested tree vertically", () => {
        const tabOne = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tabTwo = {
          tabKey: "tab2",
          name: "Tab 2",
        };
        const tabThree = {
          tabKey: "tab3",
          name: "Tab 3",
        };
        const tabFour = {
          tabKey: "tab4",
          name: "Tab 4",
        };
        const tree: Mosaic.Node = {
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            direction: "y",
            first: {
              key: 6,
              tabs: [tabTwo],
              selected: "tab2",
            },
            last: {
              key: 7,
              tabs: [tabThree, tabFour],
              selected: "tab3",
            },
          },
        };
        const nextTree = Mosaic.split(tree, tabThree.tabKey, "y");
        expect(nextTree).toEqual({
          key: 1,
          direction: "x",
          first: {
            key: 2,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 3,
            direction: "y",
            first: {
              key: 6,
              tabs: [tabTwo],
              selected: "tab2",
            },
            last: {
              key: 7,
              direction: "y",
              first: {
                key: 14,
                selected: "tab4",
                tabs: [tabFour],
              },
              last: {
                key: 15,
                selected: "tab3",
                tabs: [tabThree],
              },
            },
          },
        });
      });
    });
  });

  describe("Mosaic", () => {
    it("should render a mosaic correctly", () => {
      const tabOne = {
        tabKey: "tab1",
        name: "Tab 1",
        content: "Tab 1 Content",
      };
      const tabTwo = {
        tabKey: "tab2",
        name: "Tab 2",
        content: "Tab 2 Content",
      };
      const tabThree = {
        tabKey: "tab3",
        name: "Tab 3",
        content: "Tab 3 Content",
      };
      const tabFour = {
        tabKey: "tab4",
        name: "Tab 4",
        content: "Tab 4 Content",
      };

      const initialTree: Mosaic.Node = {
        key: 1,
        first: {
          key: 2,
          direction: "x",
          first: {
            key: 4,
            tabs: [tabOne],
            selected: "tab1",
          },
          last: {
            key: 5,
            tabs: [tabTwo],
            selected: "tab2",
          },
        },
        last: {
          key: 3,
          direction: "x",
          first: {
            key: 6,
            tabs: [tabThree],
            selected: "tab3",
          },
          last: {
            key: 7,
            tabs: [tabFour],
            selected: "tab4",
          },
        },
      };
      const { getByText } = render(<TestMosaic initialTree={initialTree} />);
      expect(getByText("Tab 1")).toBeTruthy();
      expect(getByText("Tab 2")).toBeTruthy();
      expect(getByText("Tab 3")).toBeTruthy();
      expect(getByText("Tab 4")).toBeTruthy();
    });
  });
});
