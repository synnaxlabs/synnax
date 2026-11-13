// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

import { useByKey } from "@/feature/ni/device/queries";
import { type Device } from "@/feature/ni/device/types";
import { accepts } from "@/feature/ni/task/ports";

interface ChannelSource {
  device: string;
  port?: number;
}

/** The device of the channel at the path. */
export const useChannelDevice = (
  path: deep.Target<ChannelSource>,
): Device | undefined =>
  useByKey(Form.useFieldValue(deep.widen(path).device, { optional: true }) ?? "");

/**
 * The analog input port of the channel at the path. Undefined when the device has no
 * port list or lacks the port.
 */
export const useAnalogInputPort = (
  path: deep.Target<ChannelSource>,
): ni.device.AnalogInputPort | undefined => {
  const device = useChannelDevice(path);
  const port = Form.useFieldValue(deep.widen(path).port, { optional: true });
  return device?.properties.hardware?.analogInputs.find((p) => p.port === port);
};

/**
 * The types a select offers for a channel on the port: the port's types, the current
 * type, and the types that need no port. Every type when the port or its types are
 * unknown.
 */
export const allowedTypes = <T extends string>(
  all: readonly T[],
  current: T,
  port?: { types: string[] },
  portless: T[] = [],
): T[] =>
  all.filter(
    (t) =>
      port == null || accepts(port.types, t) || t === current || portless.includes(t),
  );
