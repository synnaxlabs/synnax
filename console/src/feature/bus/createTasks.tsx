// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/bus/Form.css";

import { bus, type library, type task } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { type FC } from "react";
import { z } from "zod";

import { configureRead, configureWrite } from "@/feature/bus/configure";
import { ReadDetails, WriteDetails } from "@/feature/bus/Details";
import { Messages } from "@/feature/bus/Messages";
import { SelectLibrary } from "@/feature/bus/SelectLibrary";
import { type Accepts, validateRead, validateWrite } from "@/feature/bus/types";
import { type Command } from "@/platform/command";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const statusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish();

/** A deploy check on a config, such as {@link validateFraming}. */
export interface Check<C> {
  (ctx: z.core.ParsePayload<C>): void;
}

export interface CreateTasksParams<
  P extends string,
  R extends z.ZodType<bus.ReadConfig>,
  W extends z.ZodType<bus.WriteConfig>,
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

const createReadMessage = (entry: library.MessageEntry): bus.ReadMessage =>
  bus.readMessageZ.parse({
    message: entry.key,
    fields: entry.payload.fields.map((f) => ({ field: f.key })),
  });

const createWriteMessage = (entry: library.MessageEntry): bus.WriteMessage =>
  bus.writeMessageZ.parse({
    message: entry.key,
    fields: entry.payload.fields.map((f) => ({ field: f.key })),
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
  R extends z.ZodType<bus.ReadConfig>,
  W extends z.ZodType<bus.WriteConfig>,
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
    const cfg = readConfigZ.parse(config ?? {});
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
    const cfg = writeConfigZ.parse(config ?? {});
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

  const useCreateRead = Task.createUseCreate({
    getInitialValues: getReadInitialValues,
  });
  const useCreateWrite = Task.createUseCreate({
    getInitialValues: getWriteInitialValues,
  });

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
