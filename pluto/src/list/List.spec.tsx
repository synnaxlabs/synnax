// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type record } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { act } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { renderProp } from "@/component/renderProp";
import { List } from "@/list";
import { mockBoundingClientRect } from "@/testutil/dom";

describe("List", () => {
  interface Context {
    virtual: boolean;
    name: string;
  }
  const CONTEXTS: Context[] = [
    {
      name: "non-virtual",
      virtual: false,
    },
    {
      name: "virtual",
      virtual: true,
    },
  ];
  CONTEXTS.forEach((context) => {
    beforeAll(() => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });
    describe(context.name, () => {
      it("should render a list of items", () => {
        const result = render(
          <List.Frame data={["1", "2", "3"]} virtual={context.virtual}>
            <List.Items>
              {({ key, ...rest }: List.ItemProps<string>) => (
                <List.Item key={key} {...rest}>
                  {key}
                </List.Item>
              )}
            </List.Items>
          </List.Frame>,
        );
        expect(result.getByText("1")).toBeTruthy();
        expect(result.getByText("2")).toBeTruthy();
        expect(result.getByText("3")).toBeTruthy();
      });

      it("should allow the caller to provide a custom item getter", () => {
        const getItem = (key?: string): record.KeyedNamed<string> | undefined => {
          if (key === "1") return { key: "1", name: "one" };
          if (key === "2") return { key: "2", name: "two" };
          if (key === "3") return { key: "3", name: "three" };
          return undefined;
        };
        const result = render(
          <List.Frame<string, record.KeyedNamed<string>>
            data={["1", "2", "3"]}
            getItem={getItem}
            virtual={context.virtual}
          >
            <List.Items<string>>
              {({ itemKey }) => <div key={itemKey}>{getItem(itemKey)?.name}</div>}
            </List.Items>
          </List.Frame>,
        );
        expect(result.getByText("one")).toBeTruthy();
        expect(result.getByText("two")).toBeTruthy();
        expect(result.getByText("three")).toBeTruthy();
      });

      it("should allow the caller to pass a subscription function for whenever the item content changes", () => {
        const data: record.KeyedNamed<string>[] = [
          { key: "1", name: "one" },
          { key: "2", name: "two" },
          { key: "3", name: "three" },
        ];
        const getItem = (key?: string): record.KeyedNamed<string> | undefined => {
          if (key === "1") return data[0];
          if (key === "2") return data[1];
          if (key === "3") return data[2];
          return undefined;
        };
        const obs = new observe.Observer<void>();
        const itemProp = renderProp(({ itemKey }: List.ItemProps<string>) => {
          const item = List.useItem<string, record.KeyedNamed<string>>(itemKey);
          return <div key={itemKey}>{item?.name}</div>;
        });
        const result = render(
          <List.Frame<string, record.KeyedNamed<string>>
            data={["1", "2", "3"]}
            getItem={getItem}
            subscribe={(callback) => obs.onChange(callback)}
            virtual={context.virtual}
          >
            <List.Items<string>>{itemProp}</List.Items>
          </List.Frame>,
        );
        expect(result.getByText("one")).toBeTruthy();
        expect(result.getByText("two")).toBeTruthy();
        expect(result.getByText("three")).toBeTruthy();
        data[0] = { key: "1", name: "one-updated" };
        act(() => {
          obs.notify();
        });
        expect(result.getByText("one-updated")).toBeTruthy();
      });
    });
  });
});
