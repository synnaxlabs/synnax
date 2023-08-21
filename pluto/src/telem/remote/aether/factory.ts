// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { client } from "@/telem/client";
import { telem } from "@/telem/core";
import { Numeric } from "@/telem/remote/aether/numeric";
import { DynamicXY, XY } from "@/telem/remote/aether/xy";

type Constructor = new (key: string, client: client.Client) => telem.Telem;

const REGISTRY: Record<string, Constructor> = {
  [XY.TYPE]: XY,
  [DynamicXY.TYPE]: DynamicXY,
  [Numeric.TYPE]: Numeric,
};

export class Factory implements telem.Factory {
  private readonly client: client.Client;
  constructor(client: client.Client) {
    this.client = client;
  }

  create(key: string, props: telem.Spec): telem.Telem | null {
    if (!(props.type in REGISTRY)) return null;
    const t = new REGISTRY[props.type](key, this.client);
    t.setProps(props.props);
    return t;
  }
}
