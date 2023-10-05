// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type change, type Series } from "@synnaxlabs/x";

import { type Client as FrameClient } from "@/framer/client";
import { type Frame } from "@/framer/frame";
import { type Streamer as FrameStreamer } from "@/framer/streamer";
import { ID, type Resource } from "@/ontology/payload";
import { type Retriever } from "@/ontology/retriever";

const parseIds = (series: Series): ID[] =>
  new TextDecoder()
    .decode(series.buffer)
    .split("\n")
    .map((id) => new ID(id));

class ChangeTracker {
  private readonly obs: observe.Observer<Array<change.Change<ID, Resource>>> =
    new observe.Observer();

  private readonly streamer: FrameStreamer;
  private readonly retriever: Retriever;

  constructor(streamer: FrameStreamer, retriever: Retriever) {
    this.retriever = retriever;
    this.streamer = streamer;
    void this.start();
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
    const ids = parseIds(sets[0]);
    const resources = await this.retriever.retrieve(ids);
    this.obs.notify(
      resources.map((resource) => ({
        variant: "set",
        key: resource.id,
        value: resource,
      })),
    );
  }

  private async updateDeletes(frame: Frame): Promise<void> {
    const deletes = frame.get("sy_ontology_delete");
    if (deletes.length === 0) return;
    const ids = parseIds(deletes[0]);
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
