// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@/status/aether";
import { type telem } from "@/telem/aether";
import { NoopFactory } from "@/telem/aether/noop";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { RemoteFactory } from "@/telem/aether/remote";
import { StaticFactory } from "@/telem/aether/static";
import { TransformerFactory } from "@/telem/aether/transformers";
import { type client } from "@/telem/client";

export interface CreateOptions {
  onStatusChange?: status.Adder;
}

export interface Factory {
  type: string;
  create: (spec: telem.Spec, options?: CreateOptions) => telem.Telem | null;
}

export class CompoundFactory {
  factories: Factory[];

  type = "compound";

  constructor(factories: Factory[]) {
    this.factories = factories;
  }

  add(factory: Factory): void {
    this.factories = [
      ...this.factories.filter((f) => f.type !== factory.type),
      factory,
    ];
  }

  create(props: telem.Spec, options?: CreateOptions): telem.Telem | null {
    for (const factory of this.factories) {
      const telem = factory.create(props, options);
      if (telem != null) return telem;
    }
    return null;
  }
}

export const createFactory = (client?: client.Client): CompoundFactory => {
  const base = [new TransformerFactory(), new StaticFactory(), new NoopFactory()];
  const f = new CompoundFactory(base);
  if (client != null) f.add(new RemoteFactory(client));
  f.add(new PipelineFactory(f));
  return f;
};
