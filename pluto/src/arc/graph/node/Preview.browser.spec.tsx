// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderThemed, treatment } from "@synnaxlabs/lyra/styletest";
import { theme } from "@synnaxlabs/lyra/theme";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Node } from "@/arc/graph/node";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

const THEME = theme.themeZ.parse(theme.SYNNAX_LIGHT);

// Every element that holds text directly, so a spec checks each label of a node.
const textElements = (root: Element): Element[] => {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  const found = new Set<Element>();
  while (walker.nextNode() != null) {
    const { currentNode } = walker;
    if (currentNode.textContent?.trim() === "") continue;
    if (currentNode.parentElement != null) found.add(currentNode.parentElement);
  }
  return [...found];
};

describe("Arc graph node previews", () => {
  // A sink names its channel through a query.
  let Wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    Wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it.each(["constant", "stable_for", "add", "gt", "write"])(
    "should draw every label of a %s node in the text font",
    async (type) => {
      const spec = Node.resolveSpec(type);
      const { container } = await renderThemed(
        <Wrapper>
          <spec.Preview config={spec.defaultConfig(THEME)} />
        </Wrapper>,
      );
      const elements = textElements(container);
      expect(elements).not.toHaveLength(0);
      for (const el of elements)
        expect(treatment(el), el.textContent ?? "").toBe("prose");
    },
  );
});
