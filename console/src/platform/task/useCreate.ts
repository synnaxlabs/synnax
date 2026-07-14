// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type device,
  DisconnectedError,
  type rack,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import { Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";
import { type GetInitialValues } from "@/platform/task/Form";

export interface CreateParams {
  deviceKey?: device.Key;
  rackKey?: rack.Key;
  config?: unknown;
}

export interface CreateUseCreateParams<S extends task.Schemas = task.Schemas> {
  getInitialValues: GetInitialValues<S>;
}

export interface UseCreate {
  (): (params?: CreateParams) => void;
}

/**
 * Creates a draft task row of the given type: rack 0 unless a rack or device
 * rack is given, no start command issued. The write is untyped: a draft holds
 * a work-in-progress config, so the type's config schema is enforced by the
 * form and at deploy, not here.
 * @returns The created task.
 */
export const create = async <S extends task.Schemas = task.Schemas>({
  client,
  getInitialValues,
  deviceKey,
  rackKey,
  config,
}: CreateParams &
  CreateUseCreateParams<S> & { client: Synnax }): Promise<task.Task> => {
  // Initial values carry the zero payload's empty key; the row must mint its own.
  const { status: _, key: __, ...initial } = getInitialValues({ deviceKey, config });
  return await client.tasks.create({ ...initial, rack: rackKey ?? initial.rack ?? 0 });
};

/**
 * Builds the hook that creates a task of the given type and opens it for
 * editing. The returned callback creates a draft row and swaps the current
 * tab to the task's resource.
 */
export const createUseCreate =
  <S extends task.Schemas = task.Schemas>({
    getInitialValues,
  }: CreateUseCreateParams<S>): UseCreate =>
  () => {
    const client = PSynnax.use();
    const openTab = Panel.useOpenTab();
    const handleError = Status.useErrorHandler();
    return useCallback(
      ({ deviceKey, rackKey, config }: CreateParams = {}) => {
        handleError(async () => {
          if (client == null) throw new DisconnectedError();
          const created = await create({
            client,
            getInitialValues,
            deviceKey,
            rackKey,
            config,
          });
          openTab({ variant: "resource", resource: task.ontologyID(created.key) });
        }, "Failed to create task");
      },
      [client, openTab, handleError],
    );
  };
