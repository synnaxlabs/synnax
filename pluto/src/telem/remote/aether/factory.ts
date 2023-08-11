// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Client } from "@/telem/client";
import { TelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";
import { Numeric } from "@/telem/remote/aether/numeric";
import { DynamicXY, XY } from "@/telem/remote/aether/xy";

type Constructor = new (key: string, client: Client) => ModifiableTelemSourceMeta;

export class Factory implements TelemFactory {
  type = "range";

  private readonly client: Client;
  constructor(client: Client) {
    this.client = client;
  }

  private static readonly REGISTRY: Record<string, Constructor> = {
    [XY.TYPE]: XY,
    [DynamicXY.TYPE]: DynamicXY,
    [Numeric.TYPE]: Numeric,
  };

  create(key: string, type: string, props: any): ModifiableTelemSourceMeta | null {
    if (!(type in Factory.REGISTRY)) {
      throw new Error(`[Remote.Factory] - Unknown telemtype ${type}`);
    }
    const t = new Factory.REGISTRY[type](key, this.client);
    t.setProps(props);
    return t;
  }
}
