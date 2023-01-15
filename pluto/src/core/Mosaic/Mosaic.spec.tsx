// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Mosaic, MosaicLeaf } from ".";

import { UseMosaicProps, useMosaic } from "./useMosaic";

const TestMosaic = (props: UseMosaicProps): JSX.Element => {
  const props_ = useMosaic(props);
  return <Mosaic {...props_} />;
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
      it("shouldn't split an empty tree with one tab", () => {
        const tab = {
          tabKey: "tab1",
          name: "Tab 1",
        };
        const tree = Mosaic.insertTab({ key: 1, tabs: [] }, tab, "right", 1);
        expect(tree).toEqual({
          key: 1,
          tabs: [],
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
          direction: "horizontal",
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
        const tree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
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
          direction: "horizontal",
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
        const tree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
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
        const nextTree = Mosaic.removeTab(tree, "tab2");
        expect(nextTree).toEqual({
          key: 1,
          direction: "horizontal",
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
        const tree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
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
        const nextTree = Mosaic.removeTab(tree, "tab1");
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
        const tree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
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
        const nextTree = Mosaic.moveTab(tree, "tab2", "center", 2);
        expect(nextTree).toEqual({
          key: 1,
          direction: "horizontal",
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
      it("should correctly move a tab from the first leaf  of the root node to the second leaf", () => {
        const tabOne = {
          tabKey: "1",
          name: "Tab 1",
          content: <h1>Tab One Content</h1>,
        };
        const tabTwo = {
          tabKey: "2",
          name: "Tab 2",
          content: <h1>Tab Two Content</h1>,
        };
        const tabThree = {
          tabKey: "3",
          name: "Tab 3",
          content: <h1>Tab Three Content</h1>,
        };

        const initialTree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
          first: {
            key: 2,
            tabs: [tabOne],
          },
          last: {
            key: 3,
            tabs: [tabTwo, tabThree],
          },
        };
        const nextTree = Mosaic.moveTab(initialTree, "1", "center", 2);
        expect(nextTree).toEqual({
          key: 1,
          selected: "1",
          size: undefined,
          tabs: [tabTwo, tabThree, tabOne],
        });
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
        const tree: MosaicLeaf = {
          key: 1,
          direction: "horizontal",
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
        const nextTree = Mosaic.resizeLeaf(tree, 2, 100);
        expect(nextTree).toEqual({
          key: 1,
          direction: "horizontal",
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
        const tree: MosaicLeaf = {
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
        const tree: MosaicLeaf = {
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
  });
  describe("Mosaic", () => {
    it("should render a mosaic correctly", () => {
      const tabOne = {
        tabKey: "tab1",
        name: "Tab 1",
        content: <div>Tab 1 Content</div>,
      };
      const tabTwo = {
        tabKey: "tab2",
        name: "Tab 2",
        content: <div>Tab 2 Content</div>,
      };
      const tabThree = {
        tabKey: "tab3",
        name: "Tab 3",
        content: <div>Tab 3 Content</div>,
      };
      const tabFour = {
        tabKey: "tab4",
        name: "Tab 4",
        content: <div>Tab 4 Content</div>,
      };

      const initialTree: MosaicLeaf = {
        key: 1,
        first: {
          key: 2,
          direction: "horizontal",
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
          direction: "horizontal",
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
      expect(getByText("Tab 1 Content")).toBeTruthy();
      expect(getByText("Tab 2 Content")).toBeTruthy();
      expect(getByText("Tab 3 Content")).toBeTruthy();
      expect(getByText("Tab 4 Content")).toBeTruthy();
    });
  });
});
