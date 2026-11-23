// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  calculation,
  channel,
  DataType,
  NotFoundError,
  query,
} from "@synnaxlabs/client";
import type { Icon } from "@synnaxlabs/lyra/icon";
import { control, deep, primitive, scheduler, TimeSpan, verbs } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Calculation } from "@/calculation";
import {
  PLURAL_RESOURCE_NAME,
  RESOURCE_NAME,
  retrieveDefinition,
  retrieveMultipleDefinition,
  type RetrieveMultipleQuery,
  type RetrieveQuery,
} from "@/channel/aether/queries";
import { resolveIcon } from "@/channel/resolveIcon";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Synnax } from "@/synnax";

// The channel fields that both forms edit.
const fieldsZ = channel.payloadZ
  .pick({
    key: true,
    leaseholder: true,
    isIndex: true,
    index: true,
    virtual: true,
    concurrency: true,
  })
  .extend({
    name: channel.nameZ,
    dataType: DataType.z.transform((v) => v.toString()),
  });

export const formSchema = fieldsZ
  .extend({ requires: channel.keyZ.array().optional() })
  .refine(
    (v) => !v.isIndex || DataType.z.parse(v.dataType).equals(DataType.TIMESTAMP),
    {
      message: "Index channel must have data type TIMESTAMP",
      path: ["dataType"],
    },
  )
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual, {
    message: "Data channel must have an index",
    path: ["index"],
  });

export const formPath = deep.path<z.infer<typeof formSchema>>();

// The form edits one operation, and "none" stands for a calculation with no operations.
const formOperationZ = z.object({
  type: z.enum(["none", ...calculation.OPERATION_TYPES]),
  resetChannel: channel.keyZ,
  duration: TimeSpan.z,
});

type FormOperation = z.infer<typeof formOperationZ>;

export const calculatedFormSchema = fieldsZ.extend({
  expression: z
    .string()
    .min(1, "Expression must not be empty")
    .refine((v) => v.includes("return"), {
      message: "Expression must contain a return statement",
    }),
  operations: formOperationZ.array(),
  /** calculation is the key of the record that writes the channel, once saved. */
  calculation: calculation.keyZ.optional(),
});

export const calculatedFormPath = deep.path<z.infer<typeof calculatedFormSchema>>();

const ZERO_OPERATION: FormOperation = {
  type: "none",
  resetChannel: 0,
  duration: TimeSpan.ZERO,
};

const channelToFormValues = ({
  key,
  name,
  leaseholder,
  dataType,
  isIndex,
  index,
  virtual,
  concurrency,
}: channel.Channel): z.infer<typeof fieldsZ> => ({
  key,
  name,
  leaseholder,
  dataType: dataType.toString(),
  isIndex,
  index,
  virtual,
  concurrency,
});

export type { RetrieveMultipleQuery, RetrieveQuery } from "@/channel/aether/queries";

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  key: 0,
  name: "",
  index: 0,
  dataType: DataType.FLOAT32.toString(),
  isIndex: false,
  leaseholder: 0,
  virtual: false,
  concurrency: control.Concurrency.exclusive,
};

const ZERO_CALCULATED_FORM_VALUES: z.infer<typeof calculatedFormSchema> = {
  ...ZERO_FORM_VALUES,
  virtual: true,
  expression: "",
  operations: [ZERO_OPERATION],
  calculation: undefined,
};

export const { use, useResult, useEnsure, createSelector, createResultSelector } =
  Flux.createRetrieve<RetrieveQuery, channel.Channel>(retrieveDefinition);

export const useResultName = createResultSelector(({ name }) => name);

export const useResultIsIndex = createResultSelector(({ isIndex }) => isIndex);

export const useResultDataType = createResultSelector(
  ({ dataType }) => dataType,
  (a, b) => a.equals(b),
);

// Compared by value, so a status heartbeat is silenced.
const useResultFlags = createResultSelector(
  ({ dataType, internal, virtual }) => ({ dataType, internal, virtual }),
  (a, b) =>
    a.dataType.equals(b.dataType) &&
    a.internal === b.internal &&
    a.virtual === b.virtual,
);

/** The icon of a channel, and whether it is internal or virtual. */
export interface Kind {
  icon: Icon.FC;
  internal: boolean;
  virtual: boolean;
}

/** @returns the kind of the channel, or undefined until the channel loads. */
export const useKind = (query: RetrieveQuery): Kind | undefined => {
  const flags = useResultFlags(query).data;
  const calculated = Calculation.useIsOutput(query.key);
  if (flags == null) return undefined;
  return {
    icon: resolveIcon(flags, calculated),
    internal: flags.internal,
    virtual: flags.virtual,
  };
};

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  channel.Channel[]
>(retrieveMultipleDefinition);

const { useResult: useResultCountLimitedQuery, useRefresh: useRefreshCountLimited } =
  Flux.createRetrieve<Record<string, never>, number>({
    name: "channel count",
    retrieve: async ({ client }) => await client.channels.countLimited(),
    retryUnreachable: true,
  });

const COUNT_REFRESH_DEBOUNCE = TimeSpan.milliseconds(100);

/**
 * @returns how many channels count against the license channel limit, fetched again
 * after channel changes.
 */
export const useResultCountLimited = (): Flux.Result<number> => {
  const client = Synnax.use();
  const refresh = useRefreshCountLimited();
  useEffect(() => {
    if (client == null) return;
    // The cache keeps the count of an earlier mount, which missed the changes since.
    refresh({});
    const debounced = scheduler.debounce(() => refresh({}), COUNT_REFRESH_DEBOUNCE);
    const release = client.channels.store.subscribeBatch((events) => {
      if (events.some(query.isChange)) debounced();
    });
    return () => {
      debounced.cancel();
      release();
    };
  }, [client, refresh]);
  return useResultCountLimitedQuery({});
};

const retrieveInitialFormValues = async ({
  query: { key },
  client,
}: Flux.RetrieveParams<FormQuery>) =>
  channelToFormValues(await client.channels.retrieve(key));

const getCachedFormValues = ({
  client,
  query: { key },
}: Flux.RetrieveParams<FormQuery>) => {
  const cached = client.channels.watch({ key }).getCached();
  return query.isLive(cached) ? channelToFormValues(cached) : undefined;
};

const updateForm = async ({
  client,
  set,
  value,
  stored,
}: Flux.FormUpdateParams<typeof formSchema>) => {
  const values = value();
  if (values.requires != null) delete values.requires;
  if (primitive.isZero(values.key)) {
    const ch = await client.channels.create(values);
    set(formPath.key, ch.key);
    return;
  }
  if (stored?.name !== values.name)
    await client.channels.dispatch(values.key, channel.rename({ name: values.name }));
};

export type FormQuery = RetrieveQuery;

const formMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof formSchema
>["mountListeners"] = ({ client, query: { key }, reset }) =>
  client.channels.watch({ key }).onChange((result) => {
    if (query.isLive(result)) reset(channelToFormValues(result));
  });

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  getCached: getCachedFormValues,
  update: updateForm,
  mountListeners: formMountListeners,
});

// These convert between the form's operation and the operations of a calculation.
const toFormOperation = (op?: calculation.Operation): FormOperation => {
  if (op == null) return ZERO_OPERATION;
  if (op.type === "derivative") return { ...ZERO_OPERATION, type: "derivative" };
  return { type: op.type, duration: op.duration, resetChannel: op.reset ?? 0 };
};

const toCalculationOperations = (op?: FormOperation): calculation.Operation[] => {
  if (op == null || op.type === "none") return [];
  if (op.type === "derivative") return [{ type: "derivative" }];
  const reset = op.resetChannel === 0 ? undefined : op.resetChannel;
  return [{ type: op.type, duration: new TimeSpan(op.duration), reset }];
};

// The values of the calculated form that the calculation holds.
const toCalculationFormValues = (calc?: calculation.Calculation) => ({
  expression: calc?.expression ?? "",
  operations: [toFormOperation(calc?.operations[0])],
  calculation: calc?.key,
});

const toCalculatedFormValues = (
  ch: channel.Channel,
  calc?: calculation.Calculation,
) => ({ ...channelToFormValues(ch), ...toCalculationFormValues(calc) });

const retrieveCalculatedFormValues = async ({
  query: { key },
  client,
}: Flux.RetrieveParams<FormQuery>) => {
  const [ch, calcs] = await Promise.all([
    client.channels.retrieve(key),
    client.calculations.retrieve({ output: key }),
  ]);
  return toCalculatedFormValues(ch, calcs[0]);
};

// A change to the channel or to its calculation from elsewhere reloads the form. Each
// change carries the values of one record, so the form keeps the last values of the
// other, starting from the values it loaded with.
const calculatedFormMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof calculatedFormSchema
>["mountListeners"] = ({ client, query: { key }, reset, value }) => {
  let latest = value();
  const reload = (next: Partial<typeof latest>) => {
    latest = { ...latest, ...next };
    reset(latest);
  };
  const destructors = [
    client.channels.watch({ key }).onChange((result) => {
      if (query.isLive(result)) reload(channelToFormValues(result));
    }),
  ];
  if (latest.calculation != null)
    destructors.push(
      client.calculations.watch({ key: latest.calculation }).onChange((result) => {
        if (query.isLive(result)) reload(toCalculationFormValues(result));
      }),
    );
  return destructors;
};

const updateCalculatedForm = async ({
  client,
  set,
  value,
  stored,
}: Flux.FormUpdateParams<typeof calculatedFormSchema>) => {
  const values = value();
  const operations = toCalculationOperations(values.operations[0]);
  if (primitive.isZero(values.key)) {
    const calc = await client.calculations.create({
      name: values.name,
      expression: values.expression,
      operations,
    });
    set(calculatedFormPath.key, calc.output);
    set(calculatedFormPath.calculation, calc.key);
    return;
  }
  if (stored?.name !== values.name)
    await client.channels.dispatch(values.key, channel.rename({ name: values.name }));
  if (values.calculation == null) return;
  const edits: calculation.Action[] = [];
  if (stored?.expression !== values.expression)
    edits.push(calculation.set({ path: ["expression"], value: values.expression }));
  if (!deep.equal(toCalculationOperations(stored?.operations[0]), operations))
    edits.push(calculation.set({ path: ["operations"], value: operations }));
  if (edits.length === 0) return;
  // The form watches its calculation, so the cache drops it only on a delete.
  if (!(await client.calculations.dispatch(values.calculation, edits)))
    throw new NotFoundError("Calculation was deleted");
};

export const useCalculatedForm = Flux.createForm<
  FormQuery,
  typeof calculatedFormSchema
>({
  name: "calculated channel",
  schema: calculatedFormSchema,
  initialValues: ZERO_CALCULATED_FORM_VALUES,
  retrieve: retrieveCalculatedFormValues,
  update: updateCalculatedForm,
  mountListeners: calculatedFormMountListeners,
});

export type ListQuery = channel.RetrieveMultipleParams;

const normalizeListQuery = <Q extends ListQuery>(query: Q): Q => ({
  ...query,
  where: { internal: { eq: false }, ...query.where },
});

/** Lists the channels that are not internal, unless the query sets `where.internal`. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, channel.Channel>,
  "itemQuery" | "name"
> = {}): Flux.UseListReturn<channel.Key, channel.Channel> =>
  Flux.useList((client) => client.channels, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : normalizeListQuery(query ?? {}),
    itemQuery: (key) => ({ key }),
  });

export const { useResult: useResultList } = Flux.createRetrieve<
  ListQuery,
  channel.Channel[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: normalizeListQuery,
  ...flux.listReads(({ client, query }) => client.channels.watch(query)),
});

export interface RenameParams extends Pick<channel.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.channels.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = channel.Key | channel.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.channels.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
