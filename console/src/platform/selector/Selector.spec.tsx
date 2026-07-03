// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { type Layout } from "@/platform/layout";
import { Selector } from "@/platform/selector";

// The Selector component itself wraps its content in Pluto's Eraser, which mounts the
// aether canvas render loop and cannot run under jsdom. createSelector's factory
// contract (below) is what consumers depend on and is testable without rendering; the
// item placement behavior is covered in Item.spec.tsx.
describe("Selector.createSelector", () => {
  it("builds a renderer that forwards the selectables and text to the Selector", () => {
    const item = Selector.createSimpleItem({
      title: "Cat",
      icon: <Icon.Add />,
      layout: { type: "cat", name: "Cat", location: "mosaic" },
    });
    const Renderer = Selector.createSelector([item], "Choose one");
    expect(Renderer.displayName).toBe("LayoutSelector");
    const element = (Renderer as FC<Layout.RendererProps>)({
      layoutKey: "lk",
      visible: true,
      focused: false,
      onClose: () => {},
    }) as ReactElement<Selector.SelectorProps>;
    expect(element.type).toBe(Selector.Selector);
    expect(element.props.selectables).toEqual([item]);
    expect(element.props.text).toBe("Choose one");
    expect(element.props.layoutKey).toBe("lk");
  });
});
