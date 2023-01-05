// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { Autosize } from ".";

describe("AutoSize", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = vi.fn(
      (): DOMRect => ({
        width: 100,
        height: 200,
        top: 0,
        left: 0,
        bottom: 0,
        right: 0,
        x: 0,
        y: 0,
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        toJSON: () => {},
      })
    );
  });
  afterAll(() => {
    vi.clearAllMocks();
  });
  it("should provide a width and height to a child component", async () => {
    const c = render(
      <div style={{ height: 100, width: 100 }}>
        <Autosize style={{ height: 100, width: 100 }}>
          {({ width, height }) => {
            return (
              <div className="hello" style={{ height, width }}>
                Hello
              </div>
            );
          }}
        </Autosize>
      </div>
    );

    expect(c.getByText("Hello").style.width).toBe("100px");
  });
  it("should provide a width and height to a child element", async () => {
    const El = ({
      width,
      height,
    }: {
      width?: number;
      height?: number;
    }): JSX.Element => {
      return <div style={{ height, width }}>Hello</div>;
    };
    const c = render(
      <div style={{ height: 100, width: 100 }}>
        <Autosize style={{ height: 100, width: 100 }}>
          <El />
        </Autosize>
      </div>
    );

    expect(c.getByText("Hello").style.width).toBe("100px");
  });
});
