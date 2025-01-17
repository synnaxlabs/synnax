// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { observe } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";

import { type framer } from "@/framer";
import { type Key, type Label, labelZ, ontologyID } from "@/label/payload";
import { Retriever } from "@/label/retriever";
import { type NewLabelPayload, type SetOptions, Writer } from "@/label/writer";
import { ontology } from "@/ontology";
import { signals } from "@/signals";

const LABEL_SET_NAME = "sy_label_set";
const LABEL_DELETE_NAME = "sy_label_delete";

export class Client implements AsyncTermSearcher<string, Key, Label> {
  readonly type: string = "label";
  private readonly retriever: Retriever;
  private readonly writer: Writer;
  private readonly frameClient: framer.Client;
  private readonly ontology: ontology.Client;

  constructor(
    client: UnaryClient,
    frameClient: framer.Client,
    ontology: ontology.Client,
  ) {
    this.writer = new Writer(client);
    this.retriever = new Retriever(client);
    this.frameClient = frameClient;
    this.ontology = ontology;
  }

  async search(term: string): Promise<Label[]> {
    return await this.retriever.search(term);
  }

  async retrieve(key: Key): Promise<Label>;

  async retrieve(keys: Key[]): Promise<Label[]>;

  async retrieve(keys: Key | Key[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async retrieveFor(id: ontology.CrudeID): Promise<Label[]> {
    return await this.retriever.retrieveFor(new ontology.ID(id));
  }

  async label(
    id: ontology.CrudeID,
    labels: Key[],
    opts: SetOptions = {},
  ): Promise<void> {
    await this.writer.set(new ontology.ID(id), labels, opts);
  }

  async removeLabels(id: ontology.CrudeID, labels: Key[]): Promise<void> {
    await this.writer.remove(new ontology.ID(id), labels);
  }

  async page(offset: number, limit: number): Promise<Label[]> {
    return await this.retriever.page(offset, limit);
  }

  async create(label: NewLabelPayload): Promise<Label>;

  async create(labels: NewLabelPayload[]): Promise<Label[]>;

  async create(labels: NewLabelPayload | NewLabelPayload[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const res = await this.writer.create(labels);
    return isMany ? res : res[0];
  }

  async delete(key: Key): Promise<void>;

  async delete(keys: Key[]): Promise<void>;

  async delete(keys: Key | Key[]): Promise<void> {
    await this.writer.delete(keys);
  }

  async openChangeTracker(): Promise<signals.Observable<string, Label>> {
    return await signals.openObservable<string, Label>(
      this.frameClient,
      LABEL_SET_NAME,
      LABEL_DELETE_NAME,
      decodeChanges,
    );
  }

  async trackLabelsOf(
    id: ontology.CrudeID,
  ): Promise<observe.ObservableAsyncCloseable<Label[]>> {
    const wrapper = new observe.Observer<Label[]>();
    const initial = (await this.retrieveFor(id)).map((l) => ({
      id: ontologyID(l.key),
      key: l.key,
      name: l.name,
      data: l,
    }));
    const base = await this.ontology.openDependentTracker({
      target: new ontology.ID(id),
      dependents: initial,
      relationshipType: "labeled_by",
    });
    base.onChange((resources: ontology.Resource[]) =>
      wrapper.notify(
        resources.map((r) => ({
          key: r.id.key,
          color: r.data?.color as string,
          name: r.data?.name as string,
        })),
      ),
    );
    return wrapper;
  }
}

const decodeChanges: signals.Decoder<string, Label> = (variant, data) => {
  if (variant === "delete") return data.toUUIDs().map((v) => ({ variant, key: v }));
  return data.parseJSON(labelZ).map((l) => ({ variant, key: l.key, value: l }));
};
