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
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Select } from "@/select";
import { createRenderCounter } from "@/testutil";
import { mockGeometry } from "@/testutil/dom";
import { Triggers } from "@/triggers";

const DATA = Array.from({ length: 300 }, (_, i) => `${i}`);

const row = ({ key, ...rest }: List.ItemProps<string>): ReactElement => (
  <Select.Item key={key} {...rest}>
    {rest.itemKey}
  </Select.Item>
);

interface PinnedProps {
  pinned?: string[];
  initialHover?: number;
}

const Pinned = ({ pinned, initialHover }: PinnedProps): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame visible>
      <Select.Frame
        data={DATA}
        onChange={vi.fn()}
        initialHover={initialHover}
        pinned={pinned}
        itemHeight={33}
        virtual
      >
        <List.Scroll>
          <Select.Items<string>>{row}</Select.Items>
        </List.Scroll>
      </Select.Frame>
    </Dialog.Frame>
  </Triggers.Provider>
);

describe("Select.Frame pinned", () => {
  beforeAll(() => mockGeometry(100, 100));

  it("should keep a pinned option mounted outside the window", () => {
    const c = render(<Pinned />);
    expect(c.queryByText("150", { exact: true })).toBeNull();
    c.rerender(<Pinned pinned={["150"]} />);
    expect(c.queryByText("150", { exact: true })).not.toBeNull();
  });

  it("should keep a pinned option and the hovered option mounted together", () => {
    const c = render(<Pinned pinned={["150"]} initialHover={DATA.length - 1} />);
    expect(c.queryByText("150", { exact: true })).not.toBeNull();
    expect(c.queryByText("299", { exact: true })).not.toBeNull();
    expect(c.queryByText("151", { exact: true })).toBeNull();
  });
});

const { Counted, counts, reset } = createRenderCounter();

const OPTIONS = ["x"];

const nested = ({ key, ...rest }: List.ItemProps<string>): ReactElement => (
  <div key={key}>
    <Counted id={rest.itemKey}>
      <Select.Frame multiple data={OPTIONS} value={OPTIONS} onChange={vi.fn()}>
        <span>{rest.itemKey}</span>
      </Select.Frame>
    </Counted>
  </div>
);

const Outer = ({ data }: { data: string[] }): ReactElement => (
  <List.Frame data={data}>
    <List.Scroll>
      <List.Items<string>>{nested}</List.Items>
    </List.Scroll>
  </List.Frame>
);

describe("Select.Frame nested in a list", () => {
  it("should not render when the keys of the outer list change", () => {
    const c = render(<Outer data={["a", "b"]} />);
    reset();
    c.rerender(<Outer data={["a", "b", "c"]} />);
    expect(Object.fromEntries(counts)).toEqual({});
  });
});
