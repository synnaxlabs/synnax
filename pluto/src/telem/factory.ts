// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Telem, TelemSpec } from "@/core/vis/telem";

export interface TelemFactory {
  create: (key: string, spec: TelemSpec, root: TelemFactory) => Telem | null;
}

export class CompoundTelemFactory {
  factories: TelemFactory[];

  type = "compound";

  constructor(factories: TelemFactory[]) {
    this.factories = factories;
  }

  create(key: string, props: TelemSpec, root: TelemFactory): Telem | null {
    for (const factory of this.factories) {
      const telem = factory.create(key, props, root);
      if (telem != null) return telem;
    }
    return null;
  }
}
