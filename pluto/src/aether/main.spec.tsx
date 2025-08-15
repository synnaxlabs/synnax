// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createMockWorkers } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type FC, type PropsWithChildren, useRef } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";

export const exampleProps = z.object({
  x: z.number(),
});

class ExampleLeaf extends aether.Leaf<typeof exampleProps> {
  static readonly TYPE = "ExampleLeaf";
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  afterUpdate(): void {
    this.updatef();
  }

  afterDelete(): void {
    this.deletef();
  }
}

class ExampleComposite extends aether.Composite<typeof exampleProps, ExampleLeaf> {
  updatef = vi.fn();
  deletef = vi.fn();

  static readonly TYPE = "ExampleComposite";

  schema = exampleProps;

  afterUpdate(): void {
    this.updatef();
  }

  afterDelete(): void {
    this.deletef();
  }
}

const REGISTRY: aether.ComponentRegistry = {
  [ExampleLeaf.TYPE]: ExampleLeaf,
  [ExampleComposite.TYPE]: ExampleComposite,
};

const newProvider = async (): Promise<[FC<PropsWithChildren>, aether.Root]> => {
  const [a, b] = createMockWorkers();
  const root = aether.render({ comms: a.route("vis"), registry: REGISTRY });
  const worker = b.route<MainMessage, AetherMessage>("vis");
  return [
    (props: PropsWithChildren) => (
      <Aether.Provider worker={worker} workerKey="vis" {...props} />
    ),
    root,
  ];
};

describe("Aether Main", () => {
  describe("leaf", () => {
    it("should set the initial state correctly", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        return null;
      };
      render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(async () => root.children.length === 1).toBe(true);
      const first = root.children[0] as ExampleLeaf;
      expect(first.type).toBe(ExampleLeaf.TYPE);
      expect(first.state).toEqual({ x: 0 });
    });
    it("should update the state on a call to setState", async () => {
      const [Provider, root] = await newProvider();
      const ExampleLeafC = () => {
        const [, , setState] = Aether.use({
          type: ExampleLeaf.TYPE,
          schema: exampleProps,
          initialState: { x: 0 },
        });
        const set = useRef(false);
        if (!set.current) {
          setState({ x: 1 });
          set.current = true;
        }
        return null;
      };
      render(
        <Provider>
          <ExampleLeafC />
        </Provider>,
      );
      await expect.poll(async () => root.children.length === 1).toBe(true);
      const first = root.children[0] as ExampleLeaf;
      expect(first.type).toBe(ExampleLeaf.TYPE);
      expect(first.state).toEqual({ x: 1 });
    });
  });
});
