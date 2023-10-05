// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type change, type Destructor } from "@synnaxlabs/x";

import { type Client as FrameClient } from "@/framer/client";
import { type Frame } from "@/framer/frame";
import { type Streamer as FrameStreamer } from "@/framer/streamer";
import { ID, type Resource } from "@/ontology/payload";
import { type Retriever } from "@/ontology/retriever";

import { QueryError } from "..";

export type Change = change.Change<ID, Resource>;

export const parseIDsFromBuffer = (buf: ArrayBufferLike): ID[] =>
  new TextDecoder()
    .decode(buf)
    .split("\n")
    .slice(0, -1)
    .map((id) => new ID(id));

export class ChangeTracker implements observe.Observable<Change[]> {
  private readonly obs: observe.Observer<Change[]> = new observe.Observer();

  private readonly streamer: FrameStreamer;
  private readonly retriever: Retriever;
  private readonly closePromise: Promise<void>;

  constructor(streamer: FrameStreamer, retriever: Retriever) {
    this.retriever = retriever;
    this.streamer = streamer;
    this.closePromise = this.start();
  }

  onChange(handler: observe.Handler<Change[]>): Destructor {
    return this.obs.onChange(handler);
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async start(): Promise<void> {
    for await (const frame of this.streamer) {
      await this.update(frame);
    }
  }

  private async update(frame: Frame): Promise<void> {
    await this.updateSets(frame);
    await this.updateDeletes(frame);
  }

  private async updateSets(frame: Frame): Promise<void> {
    const sets = frame.get("sy_ontology_set");
    if (sets.length === 0) return;
    // We should only ever get one series of sets
    const ids = parseIDsFromBuffer(sets[0].buffer);
    try {
      const resources = await this.retriever.retrieve(ids);
      this.obs.notify(
        resources.map((resource) => ({
          variant: "set",
          key: resource.id,
          value: resource,
        })),
      );
    } catch (e) {
      if (e instanceof QueryError) {
        console.warn(e);
        return;
      }
      throw e;
    }
  }

  private async updateDeletes(frame: Frame): Promise<void> {
    const deletes = frame.get("sy_ontology_delete");
    if (deletes.length === 0) return;
    // We should only ever get one series of deletes
    const ids = parseIDsFromBuffer(deletes[0].buffer);
    this.obs.notify(
      ids.map((id) => ({
        variant: "delete",
        key: id,
      })),
    );
  }

  static async open(client: FrameClient, retriever: Retriever): Promise<ChangeTracker> {
    const streamer = await client.newStreamer([
      "sy_ontology_set",
      "sy_ontology_delete",
    ]);
    return new ChangeTracker(streamer, retriever);
  }
}
