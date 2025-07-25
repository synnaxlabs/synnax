// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, type record } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod/v4";

import { framer } from "@/framer";
import {
  type Device,
  deviceZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type State,
  stateZ,
} from "@/hardware/device/payload";
import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import { ontology } from "@/ontology";
import { signals } from "@/signals";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const SET_CHANNEL_NAME = "sy_device_set";
const DELETE_CHANNEL_NAME = "sy_device_delete";
const STATE_CHANNEL_NAME = "sy_device_state";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";
const CREATE_ENDPOINT = "/hardware/device/create";
const DELETE_ENDPOINT = "/hardware/device/delete";

const createReqZ = z.object({ devices: newZ.array() });

const createResZ = z.object({ devices: deviceZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
  models: z.string().array().optional(),
  locations: z.string().array().optional(),
  racks: rackKeyZ.array().optional(),
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  ignoreNotFound: z.boolean().optional(),
  includeState: z.boolean().optional(),
});

interface RetrieveRequest extends z.input<typeof retrieveReqZ> {}

export interface RetrieveOptions
  extends Pick<
    RetrieveRequest,
    "limit" | "offset" | "makes" | "ignoreNotFound" | "includeState"
  > {}

interface PageOptions extends Pick<RetrieveOptions, "makes"> {}

const retrieveResZ = z.object({ devices: nullableArrayZ(deviceZ) });

export class Client implements AsyncTermSearcher<string, Key, Device> {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
    StateDetails extends {} = record.Unknown,
  >(
    key: string,
    options?: RetrieveOptions,
  ): Promise<Device<Properties, Make, Model, StateDetails>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
    StateDetails extends {} = record.Unknown,
  >(
    keys: string[],
    options?: RetrieveOptions,
  ): Promise<Array<Device<Properties, Make, Model, StateDetails>>>;

  async retrieve<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
    StateDetails extends {} = record.Unknown,
  >(
    keys: string | string[],
    options?: RetrieveOptions,
  ): Promise<
    | Device<Properties, Make, Model, StateDetails>
    | Array<Device<Properties, Make, Model, StateDetails>>
  > {
    const isSingle = !Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: array.toArray(keys), ...options },
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Device", keys, res.devices, isSingle);
    return isSingle
      ? (res.devices[0] as Device<Properties, Make, Model, StateDetails>)
      : (res.devices as Array<Device<Properties, Make, Model, StateDetails>>);
  }

  async search(term: string, options?: RetrieveOptions): Promise<Device[]> {
    return (
      await sendRequired(
        this.client,
        RETRIEVE_ENDPOINT,
        { search: term, ...options },
        retrieveReqZ,
        retrieveResZ,
      )
    ).devices;
  }

  async page(offset: number, limit: number, options?: PageOptions): Promise<Device[]> {
    return (
      await sendRequired(
        this.client,
        RETRIEVE_ENDPOINT,
        { offset, limit, ...options },
        retrieveReqZ,
        retrieveResZ,
      )
    ).devices;
  }

  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(device: New<Properties, Make>): Promise<Device<Properties, Make, Model>>;
  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(devices: New<Properties, Make>[]): Promise<Device<Properties, Make, Model>[]>;
  async create<
    Properties extends record.Unknown = record.Unknown,
    Make extends string = string,
    Model extends string = string,
  >(
    devices: New<Properties, Make> | New<Properties, Make>[],
  ): Promise<Device<Properties, Make, Model> | Device<Properties, Make, Model>[]> {
    const isSingle = !Array.isArray(devices);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { devices: array.toArray(devices) },
      createReqZ,
      createResZ,
    );
    return isSingle
      ? (res.devices[0] as Device<Properties, Make, Model>)
      : (res.devices as Device<Properties, Make, Model>[]);
  }

  async delete(keys: string | string[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async openDeviceTracker(): Promise<signals.Observable<string, Device>> {
    return await signals.openObservable<string, Device>(
      this.frameClient,
      SET_CHANNEL_NAME,
      DELETE_CHANNEL_NAME,
      decodeDeviceChanges,
    );
  }

  async openStateObserver<Details extends {} = record.Unknown>(): Promise<
    framer.ObservableStreamer<State<Details>[]>
  > {
    return new framer.ObservableStreamer<State<Details>[]>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const states = s.parseJSON(stateZ);
        return [states as State<Details>[], true];
      },
    );
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, Key, Device> {
    return {
      type: this.type,
      search: async (term: string) => await this.search(term, options),
      retrieve: async (keys: string[]) => await this.retrieve(keys, options),
      page: async (offset: number, limit: number) =>
        await this.page(offset, limit, options),
    };
  }
}

const decodeDeviceChanges: signals.Decoder<string, Device> = (variant, data) => {
  if (variant === "delete")
    return data.toStrings().map((k) => ({ variant, key: k, value: undefined }));
  return data.parseJSON(deviceZ).map((d) => ({ variant, key: d.key, value: d }));
};

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
