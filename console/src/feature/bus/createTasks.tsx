// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/bus/Form.css";

import {
  bus,
  DisconnectedError,
  type library,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { z } from "zod";

import { configureRead, configureWrite } from "@/feature/bus/configure";
import { ReadDetails, WriteDetails } from "@/feature/bus/Details";
import { Messages } from "@/feature/bus/Messages";
import { SelectLibrary } from "@/feature/bus/SelectLibrary";
import {
  type Accepts,
  type ReadConfig,
  validateRead,
  validateWrite,
  type WriteConfig,
} from "@/feature/bus/types";
import { type Command } from "@/platform/command";
import { Panel } from "@/platform/panel";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const statusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

/** A deploy check on a config, such as {@link validateFraming}. */
export interface Check<C> {
  (ctx: z.core.ParsePayload<C>): void;
}

export interface CreateTasksParams<
  P extends string,
  R extends z.ZodType<ReadConfig>,
  W extends z.ZodType<WriteConfig>,
> {
  /** The task type prefix, such as can. */
  prefix: P;
  /** The integration's name in titles, such as CAN. */
  name: string;
  icon: Icon.ReactElement;
  readConfigZ: R;
  writeConfigZ: W;
  /** Whether the integration can carry a library message. */
  accepts: Accepts;
  /** Selects the task's device. */
  SelectDevice: FC;
  /** Settings a read task carries beyond the shared ones, such as framing. */
  ReadSettings?: FC;
  /** Settings a write task carries beyond the shared ones, such as framing. */
  WriteSettings?: FC;
  /** Deploy checks a read config needs beyond the shared ones. */
  readChecks?: Check<z.infer<R>>[];
  /** Deploy checks a write config needs beyond the shared ones. */
  writeChecks?: Check<z.infer<W>>[];
}

/**
 * @returns the library a new task starts from. The Core rejects a task config that
 * names no library, so a draft needs one before it exists.
 * @throws {Error} if no library exists.
 */
const retrieveDefaultLibrary = async (
  client: Client,
  name: string,
): Promise<library.Key> => {
  const [first] = await client.libraries.retrieve({ limit: 1 });
  if (first == null) throw new Error(`Create a library before creating a ${name} task`);
  return first.key;
};

const createUseCreate =
  (getInitialValues: Task.GetInitialValues, name: string): Task.UseCreate =>
  ({ tabKey } = {}) => {
    const client = PSynnax.use();
    const openTab = Panel.useOpenTab();
    const handleError = Status.useErrorHandler();
    return useCallback(
      ({ deviceKey, rackKey, config }: Task.CreateParams = {}) =>
        handleError(async () => {
          if (client == null) throw new DisconnectedError();
          const created = await Task.create({
            client,
            getInitialValues,
            deviceKey,
            rackKey,
            config: config ?? { library: await retrieveDefaultLibrary(client, name) },
          });
          openTab({
            variant: "resource",
            resource: task.ontologyID(created.key),
            key: tabKey,
          });
        }, `Failed to create ${name} task`),
      [client, openTab, handleError, tabKey],
    );
  };

const createReadMessage = (entry: library.MessageEntry): bus.ReadMessage =>
  bus.readMessageZ.parse({
    message: entry.key,
    fields: entry.fields.map((f) => ({ field: f.key })),
  });

const createWriteMessage = (entry: library.MessageEntry): bus.WriteMessage =>
  bus.writeMessageZ.parse({
    message: entry.key,
    fields: entry.fields.map((f) => ({ field: f.key })),
  });

const readDetails = Component.renderProp(ReadDetails);
const writeDetails = Component.renderProp(WriteDetails);

const applyChecks = <S extends z.ZodType>(schema: S, checks: Check<z.infer<S>>[]): S =>
  checks.reduce((s, check) => s.check(check), schema);

/**
 * Creates the read and write task forms of a bus integration and everything that
 * registers them. An integration supplies its config schemas, its device select, and
 * any settings beyond the library, message, and field bindings every bus task shares.
 */
export const createTasks = <
  P extends string,
  R extends z.ZodType<ReadConfig>,
  W extends z.ZodType<WriteConfig>,
>({
  prefix,
  name,
  icon,
  readConfigZ,
  writeConfigZ,
  accepts,
  SelectDevice,
  ReadSettings,
  WriteSettings,
  readChecks = [],
  writeChecks = [],
}: CreateTasksParams<P, R, W>) => {
  const READ_TYPE = `${prefix}_read` as const;
  const WRITE_TYPE = `${prefix}_write` as const;
  const READ_SCHEMAS = {
    type: z.literal(READ_TYPE),
    config: readConfigZ,
    statusData: statusDataZ,
  } as const satisfies task.Schemas;
  const WRITE_SCHEMAS = {
    type: z.literal(WRITE_TYPE),
    config: writeConfigZ,
    statusData: statusDataZ,
  } as const satisfies task.Schemas;
  type ReadSchemas = typeof READ_SCHEMAS;
  type WriteSchemas = typeof WRITE_SCHEMAS;

  const Shared = () => (
    <>
      <SelectDevice />
      <SelectLibrary />
      <Flex.Box x grow>
        <Task.Fields.DataSaving />
        <Task.Fields.AutoStart />
      </Flex.Box>
    </>
  );

  const ReadProperties = () => (
    <>
      <Shared />
      {ReadSettings != null && <ReadSettings />}
    </>
  );

  const WriteProperties = () => (
    <>
      <Shared />
      {WriteSettings != null && <WriteSettings />}
    </>
  );

  const ReadForm = () => (
    <Messages accepts={accepts} create={createReadMessage} details={readDetails} />
  );

  const WriteForm = () => (
    <Messages accepts={accepts} create={createWriteMessage} details={writeDetails} />
  );

  const getReadInitialValues: Task.GetInitialValues<ReadSchemas> = ({
    deviceKey,
    config,
  }) => {
    const cfg = readConfigZ.parse(config ?? { library: uuid.ZERO });
    return {
      name: `${name} read task`,
      type: READ_TYPE,
      config: { ...cfg, device: deviceKey ?? cfg.device },
    };
  };

  const getWriteInitialValues: Task.GetInitialValues<WriteSchemas> = ({
    deviceKey,
    config,
  }) => {
    const cfg = writeConfigZ.parse(config ?? { library: uuid.ZERO });
    return {
      name: `${name} write task`,
      type: WRITE_TYPE,
      config: { ...cfg, device: deviceKey ?? cfg.device },
    };
  };

  const Read = Task.wrapForm({
    Properties: ReadProperties,
    Form: ReadForm,
    schemas: READ_SCHEMAS,
    deployConfigZ: applyChecks(readConfigZ, [validateRead, ...readChecks]),
    type: READ_TYPE,
    getInitialValues: getReadInitialValues,
    onConfigure: configureRead,
  });

  const Write = Task.wrapForm({
    Properties: WriteProperties,
    Form: WriteForm,
    schemas: WRITE_SCHEMAS,
    deployConfigZ: applyChecks(writeConfigZ, [validateWrite, ...writeChecks]),
    type: WRITE_TYPE,
    getInitialValues: getWriteInitialValues,
    onConfigure: configureWrite,
  });

  const useCreateRead = createUseCreate(getReadInitialValues, `${name} read`);
  const useCreateWrite = createUseCreate(getWriteInitialValues, `${name} write`);

  const COMMANDS: Command.Command[] = [
    Task.createCommand({
      key: `${prefix}_create_read_task`,
      name: `Create ${name} read task`,
      icon,
      useOnSelect: useCreateRead,
    }),
    Task.createCommand({
      key: `${prefix}_create_write_task`,
      name: `Create ${name} write task`,
      icon,
      useOnSelect: useCreateWrite,
    }),
  ];

  const SELECTABLES: Selector.Selectable[] = [
    Selector.createSelectable({
      type: READ_TYPE,
      title: `${name} read task`,
      icon,
      useOnSelect: useCreateRead,
    }),
    Selector.createSelectable({
      type: WRITE_TYPE,
      title: `${name} write task`,
      icon,
      useOnSelect: useCreateWrite,
    }),
  ];

  const FORMS: Task.Forms = { [READ_TYPE]: Read, [WRITE_TYPE]: Write };

  return {
    PREFIX: prefix,
    READ_TYPE,
    WRITE_TYPE,
    READ_SCHEMAS,
    WRITE_SCHEMAS,
    Read,
    Write,
    useCreateRead,
    useCreateWrite,
    COMMANDS,
    SELECTABLES,
    FORMS,
  };
};
