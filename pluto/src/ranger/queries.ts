// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, ranger } from "@synnaxlabs/client";
import { z } from "zod";

import { Warp } from "@/warp";

const RANGE_SET_CHANNEL = "sy_range_set";
const RANGE_DELETE_CHANNEL = "sy_range_delete";

const decodeRanges: Warp.Decoder<ranger.Payload | null> = async ({
  fr,
  channels,
  client,
  current,
}) => {
  if (current == null) return [null, false];
  const [setChannel, deleteChannel] = channels;
  const sets = client.ranges.sugarMany(
    fr.get(setChannel.key).parseJSON(ranger.payloadZ),
  );
  const deletes = Array.from(fr.get(deleteChannel.key).as("string"));
  if (deletes.includes(current.key)) return [null, true];
  if (sets.some((r) => r.key === current.key)) return [current, true];
  return [null, false];
};

const retrieveChannels: Warp.RetrieveChannels = async ({ client }) =>
  await client.channels.retrieve([RANGE_SET_CHANNEL, RANGE_DELETE_CHANNEL]);

const retrieveSingle: (key?: string) => Warp.Retrieve<ranger.Payload | null> =
  (key) =>
  async ({ client }) => {
    if (key == null) return null;
    return (await client.ranges.retrieve(key)).payload;
  };

export const useRetrieve = <T extends ranger.Payload = ranger.Range>(
  key?: string,
  initialValue?: T,
): Warp.UseRetrieveReturn<T> =>
  Warp.useRetrieve({
    queryKey: [key],
    initialValue: initialValue ?? null,
    retrieveChannels,
    decode: decodeRanges,
    retrieve: retrieveSingle(key),
  }) as Warp.UseRetrieveReturn<T>;

export interface UseSyncFormArgs
  extends Pick<Warp.UseFormProps<typeof ranger.payloadZ>, "values" | "name"> {
  key: string;
}

export const useSyncedForm = ({ key, ...args }: UseSyncFormArgs) =>
  Warp.useForm({
    ...args,
    queryKey: [key],
    name: "Range",
    retrieve: retrieveSingle(key),
    retrieveChannels,
    decode: decodeRanges,
    applyChanges: async ({ client, values }) => {
      await client.ranges.create(values);
    },
    applyObservable: ({ changes, ctx }) => ctx.set("", changes),
  });

export const useRetrieveParent = (
  key?: ranger.Key,
): Warp.UseRetrieveReturn<ranger.Range> =>
  Warp.useRetrieve({
    queryKey: [key],
    initialValue: null,
    retrieve: async ({ client }) => {
      if (key == null) return null;
      return await client.ranges.retrieveParent(key);
    },
    retrieveChannels: async ({ client }) =>
      await client.channels.retrieve([RANGE_SET_CHANNEL, RANGE_DELETE_CHANNEL]),
    decode: decodeRanges,
  }) as Warp.UseRetrieveReturn<ranger.Range>;

const decodeLabels: Warp.Decoder<label.Label[]> = async ({
  fr,
  channels,
  client,
  current,
}) => {
  if (current == null) return [null, false];
  const target = new ontology.ID({ type: "range", key: "123" });
  const [setChannel, setRelationshipChannel, deleteRelationshipChannel] = channels;
  const labelSets = fr.get(setChannel.key).parseJSON(label.labelZ);
  let relationshipChanges = ontology.parseRelationshipChange(
    fr,
    setRelationshipChannel.key,
    deleteRelationshipChannel.key,
  );
  relationshipChanges = ontology.filterRelationshipChanges(relationshipChanges, {
    target,
    relDir: "from",
    relType: "labeled_by",
    resourceType: "label",
  });
  const relationshipDeletes = relationshipChanges.filter(
    ({ variant }) => variant === "delete",
  );
  const relationshipSets = relationshipChanges.filter(
    ({ variant }) => variant === "set",
  );
  const newLabels = await client.labels.retrieve(
    relationshipSets.map(({ key }) => key.to.key),
  );
  current = current.filter(
    (l) => !relationshipDeletes.some((r) => r.key.to.key === l.key),
  );
  current = [...current, ...newLabels];
  labelSets.forEach((l, i) => {
    const existing = current.find((c) => c.key === l.key);
    if (existing == null) current[i] = l;
  });
  return [current, true];
};

const retrieveLabels =
  (key?: ranger.Key): Warp.Retrieve<label.Label[]> =>
  async ({ client }) => {
    if (key == null) return [];
    return await client.labels.retrieveFor(new ontology.ID({ type: "range", key }));
  };

const retrieveLabelsChannels: Warp.RetrieveChannels = async ({ client }) =>
  await client.channels.retrieve([
    "sy_label_set",
    "sy_relationship_set",
    "sy_relationship_delete",
  ]);

export const useRetrieveLabels = (
  key?: ranger.Key,
): Warp.UseRetrieveReturn<label.Label[], label.Label[]> =>
  Warp.useRetrieve<label.Label[], label.Label[]>({
    queryKey: [key],
    initialValue: [],
    retrieve: retrieveLabels(key),
    retrieveChannels: retrieveLabelsChannels,
    decode: decodeLabels,
  });

const labelsFormSchema = z.object({
  labels: z.array(z.string()),
});

export interface UseSyncedLabelsFormArgs
  extends Pick<Warp.UseFormProps<typeof labelsFormSchema>, "values"> {
  key: string;
}

export const useSyncedLabelsForm = ({ key, ...args }: UseSyncedLabelsFormArgs) =>
  Warp.useForm({
    ...args,
    queryKey: ["range", key, "labels"],
    name: "Labels",
    schema: labelsFormSchema,
    retrieve: async (args) => {
      const labels = await retrieveLabels(key)(args);
      return { labels: labels.map((l) => l.key) };
    },
    retrieveChannels: retrieveLabelsChannels,
    // @ts-expect-error - dog
    decode: decodeLabels,
    applyChanges: async ({ client, values }) => {
      await client.labels.label(
        new ontology.ID({ type: "range", key }),
        values.labels,
        { replace: true },
      );
    },
    applyObservable: ({ changes, ctx }) => ctx.set("", changes),
  });
