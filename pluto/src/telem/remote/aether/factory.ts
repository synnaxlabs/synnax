// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type client } from "@/telem/client";
import { type telem } from "@/telem/core";
import { NumericSource } from "@/telem/remote/aether/numeric";
import { DynamicSeriesSource, SeriesSource } from "@/telem/remote/aether/series";

type Constructor = new (client: client.Client, props: unknown) => telem.Telem;

const REGISTRY: Record<string, Constructor> = {
  [SeriesSource.TYPE]: SeriesSource,
  [DynamicSeriesSource.TYPE]: DynamicSeriesSource,
  [NumericSource.TYPE]: NumericSource,
};

export class Factory implements telem.Factory {
  private readonly client: client.Client;
  constructor(client: client.Client) {
    this.client = client;
  }

  create(spec: telem.Spec): telem.Telem | null {
    if (!(spec.type in REGISTRY)) return null;
    const t = new REGISTRY[spec.type](this.client, spec.props);
    return t;
  }
}
