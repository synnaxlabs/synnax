// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type telem } from "@/telem/aether";
import { NoopFactory } from "@/telem/aether/noop";
import { PipelineFactory } from "@/telem/aether/pipeline";
import { RemoteFactory } from "@/telem/aether/remote";
import { StaticFactory } from "@/telem/aether/static";
import { TransformerFactory } from "@/telem/aether/transformers";
import { type client } from "@/telem/client";

export interface Factory {
  type: string;
  create: (spec: telem.Spec) => telem.Telem | null;
}

export class CompoundTelemFactory {
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

  create(props: telem.Spec): telem.Telem | null {
    for (const factory of this.factories) {
      const telem = factory.create(props);
      if (telem != null) return telem;
    }
    return null;
  }
}

export const factory = (client?: client.Client): CompoundTelemFactory => {
  const base = [new TransformerFactory(), new StaticFactory(), new NoopFactory()];
  const f = new CompoundTelemFactory(base);
  if (client != null) f.add(new RemoteFactory(client));
  f.add(new PipelineFactory(f));
  return f;
};
