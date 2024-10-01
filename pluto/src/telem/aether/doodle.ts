// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, Series } from "@synnaxlabs/x";
import { z } from "zod";

import { convertSeriesFloat32 } from "@/telem/aether/convertSeries";
import {
  AbstractSource,
  type SeriesSourceSpec,
  type Spec,
  type Telem,
} from "@/telem/aether/telem";
import { type client } from "@/telem/client";

const channelDataSourcePropsZ = z.object({
  channel: z.string(),
});

export type DoodleDataProps = z.input<typeof channelDataSourcePropsZ>;

// ChannelData reads a fixed time range of data from a particular channel or its index.
export class DoodleData
  extends AbstractSource<typeof channelDataSourcePropsZ>
  implements DoodleData
{
  static readonly TYPE = "doodle-source";
  private readonly client: client.ReadClient & client.ChannelClient;
  private data: Series[] = [];
  private valid: boolean = false;
  schema = channelDataSourcePropsZ;

  constructor(client: client.ReadClient & client.ChannelClient, props: unknown) {
    super(props);
    this.client = client;
  }

  async cleanup(): Promise<void> {
    this.data.forEach((d) => d.release());
    this.valid = false;
  }

  async value(): Promise<[bounds.Bounds, Series[]]> {
    if (!this.valid) {
      const [bounds, data] = await this.readFixed();
      this.data = data;
      this.valid = true;
      return [bounds, data];
    }
    return [this.data[0].bounds, this.data];
  }

  async readFixed(): Promise<[bounds.Bounds, Series[]]> {
    console.log("CH", this.props.channel);
    if (this.props.channel === "0") this.props.channel = "Time";
    const result = await fetch("http://127.0.0.1:5000/api/v1/value", {
      method: "POST",
      body: JSON.stringify({ channel: this.props.channel }),
      headers: {
        "Content-Type": "application/json",
      },
    });
    const json = await result.json();
    let dataType = "float32";
    let data = json.value;
    if (this.props.channel === "Time") {
      dataType = "timestamp";
      data = data.map((d: number) => BigInt(d));
    }
    console.log("RES", json.result);
    try {
      const s = new Series({ dataType, data });
      const f32 = convertSeriesFloat32(s);
      console.log(Array.from(f32));
      return [s.bounds, [f32]];
    } catch (e) {
      console.error(e);
      return [bounds.ZERO, []];
    }
  }
}

type Constructor = new (client: client.Client, props: unknown) => Telem;

const REGISTRY: Record<string, Constructor> = {
  [DoodleData.TYPE]: DoodleData,
};

export class DoodleFactory implements DoodleFactory {
  type = "doodle";
  private readonly client: client.Client;
  constructor(client: client.Client) {
    this.client = client;
  }

  create(spec: Spec): Telem | null {
    const V = REGISTRY[spec.type];
    if (V == null) return null;
    return new V(this.client, spec.props);
  }
}

export const doodleData = (props: DoodleDataProps): SeriesSourceSpec => ({
  type: DoodleData.TYPE,
  props,
  variant: "source",
  valueType: "series",
});
