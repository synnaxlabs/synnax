// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Diagram } from "@/components/diagram";

const ROOT: Diagram.NestNode = {
  key: "project",
  label: "Project",
  columns: 2,
  narrowColumns: 1,
  children: [
    { key: "short", label: "A" },
    { key: "long", label: "A much longer label" },
  ],
};

const leaves = (variant: "wide" | "narrow") => {
  document.body.innerHTML = renderToString(<Diagram.Nest root={ROOT} title="Nest" />);
  return [
    ...document.querySelectorAll(
      `.block-diagram__svg--${variant} .block-diagram__node`,
    ),
  ].map((leaf) => ({
    x: Number(leaf.getAttribute("x")),
    y: Number(leaf.getAttribute("y")),
    width: Number(leaf.getAttribute("width")),
  }));
};

describe("Nest", () => {
  it("should give every leaf the width of the widest leaf", () => {
    const [short, long] = leaves("wide");
    expect(short.width).toEqual(long.width);
  });

  it("should place children side by side in the given columns", () => {
    const [short, long] = leaves("wide");
    expect(long.y).toEqual(short.y);
    expect(long.x).toBeGreaterThan(short.x + short.width);
  });

  it("should stack children in the narrow columns of the narrow layout", () => {
    const [short, long] = leaves("narrow");
    expect(long.x).toEqual(short.x);
    expect(long.y).toBeGreaterThan(short.y);
  });
});
