// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { toArray } from "@synnaxlabs/x";

import { Warp } from "@/warp";

const LABEL_SET_CHANNEL = "sy_label_set";
const LABEL_DELETE_CHANNEL = "sy_label_delete";

const decodeLabels: Warp.Decoder<label.Label | label.Label[], label.Label[]> = async ({
  fr,
  channels,
  current,
}) => {
  const curr = toArray(current ?? []);
  const [setChannel, deleteChannel] = channels;
  console.log(fr.get(setChannel.key));
  const sets = fr.get(setChannel.key).parseJSON(label.labelZ);
  console.log(fr.get(deleteChannel.key));
  const deletes = fr.get(deleteChannel.key).series.flatMap((s) => s.toUUIDs());
  const setKeys = new Set(sets.map((s) => s.key));
  console.log(curr, sets, deletes);
  return [
    [
      ...curr.filter((l) => !deletes.some((d) => d === l.key) && !setKeys.has(l.key)),
      ...sets,
    ],
    true,
  ];
};

const retrieveLabelsChannels: Warp.RetrieveChannels = async ({ client }) =>
  await client.channels.retrieve([LABEL_SET_CHANNEL, LABEL_DELETE_CHANNEL]);

const retrieveLabels =
  (): Warp.Retrieve<label.Label[]> =>
  async ({ client }) =>
    await client.labels.page(0, 100);

const retrieveLabel =
  (key: label.Key): Warp.Retrieve<label.Label> =>
  async ({ client }) =>
    await client.labels.retrieve(key);

export const useRetrieveMany = (): Warp.UseRetrieveReturn<
  label.Label[],
  label.Label[]
> =>
  Warp.useRetrieve<label.Label[], label.Label[]>({
    queryKey: [],
    initialValue: [],
    retrieve: retrieveLabels(),
    retrieveChannels: retrieveLabelsChannels,
    decode: decodeLabels,
  });

export const useRetrieve = (key: label.Key): Warp.UseRetrieveReturn<label.Label> =>
  Warp.useRetrieve<label.Label>({
    queryKey: [key],
    initialValue: undefined,
    retrieve: retrieveLabel(key),
    retrieveChannels: retrieveLabelsChannels,
    decode: async (args) => {
      const [current, valid] = await decodeLabels(args);
      if (!valid || current == null) return [null, false];
      return [current[0], true];
    },
  });

export interface UseSyncedFormArgs
  extends Pick<Warp.UseFormProps<typeof label.labelZ>, "values" | "autoSave"> {
  key: label.Key;
}

export const useSyncedForm = ({ key, ...args }: UseSyncedFormArgs) =>
  Warp.useForm({
    ...args,
    queryKey: [key],
    name: "Label",
    retrieve: retrieveLabel(key),
    retrieveChannels: retrieveLabelsChannels,
    decode: decodeLabels,
    applyChanges: async ({ client, values }) => {
      await client.labels.create(values);
    },
  });
