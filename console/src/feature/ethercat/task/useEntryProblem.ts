// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { entryProblem } from "@/feature/ethercat/task/configure";
import { type Channel, type PDOType } from "@/feature/ethercat/task/types";

/** The path of the channels of an EtherCAT task in its form. */
export const CHANNELS_PATH = deep.path<{ config: { channels: Channel[] } }>().config
  .channels;

/**
 * @returns why Start would refuse the channel of the task in the form, or null when it
 * would not, when the channel is disabled, or while the slave loads.
 */
export const useEntryProblem = (
  ch: Channel,
  slave: SlaveDevice | undefined,
  pdoType: PDOType,
): string | null => {
  const channels = PForm.useFieldValue(CHANNELS_PATH);
  if (slave == null || ch.disabled) return null;
  const enabled = channels.filter((o) => !o.disabled && o.device === ch.device);
  return entryProblem(slave, enabled, ch, pdoType);
};
