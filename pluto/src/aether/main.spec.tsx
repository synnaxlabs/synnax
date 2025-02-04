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
import { type MainMessage, type WorkerMessage } from "@/aether/message";

export const exampleProps = z.object({
  x: z.number(),
});

class ExampleLeaf extends aether.Leaf<typeof exampleProps> {
  static readonly TYPE = "ExampleLeaf";
  updatef = vi.fn();
  deletef = vi.fn();

  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef();
  }

  async afterDelete(): Promise<void> {
    this.deletef();
  }
}

class ExampleComposite extends aether.Composite<typeof exampleProps, ExampleLeaf> {
  updatef = vi.fn();
  deletef = vi.fn();

  static readonly TYPE = "ExampleComposite";

  schema = exampleProps;

  async afterUpdate(): Promise<void> {
    this.updatef();
  }

  async afterDelete(): Promise<void> {
    this.deletef();
  }
}

const REGISTRY: aether.ComponentRegistry = {
  [ExampleLeaf.TYPE]: ExampleLeaf,
  [ExampleComposite.TYPE]: ExampleComposite,
};

const newProvider = async (): Promise<[FC<PropsWithChildren>, aether.Root]> => {
  const [a, b] = createMockWorkers();
  const root = await aether.render({ comms: a.route("vis"), registry: REGISTRY });
  const worker = b.route<MainMessage, WorkerMessage>("vis");
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
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(root.children).toHaveLength(1);
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
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(root.children).toHaveLength(1);
      const first = root.children[0] as ExampleLeaf;
      expect(first.type).toBe(ExampleLeaf.TYPE);
      expect(first.state).toEqual({ x: 1 });
    });
  });
});
