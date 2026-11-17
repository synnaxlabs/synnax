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
import { deep, primitive, unique } from "@synnaxlabs/x";
import { useRef, useSyncExternalStore } from "react";

import { useByKey, useByKeys } from "@/feature/ni/device/queries";
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
): ni.AnalogInputPort | undefined => {
  const device = useChannelDevice(path);
  const port = Form.useFieldValue(deep.widen(path).port, { optional: true });
  return device?.properties.hardware?.analogInputs.find((p) => p.port === port);
};

/**
 * The analog input port of each channel that the field at the path edits, which is
 * every channel under a `Form.Fanout`. An entry is undefined while its device loads, or
 * when the device has no port list or lacks the port.
 */
export const useAnalogInputPorts = (
  path: deep.Target<ChannelSource>,
): (ni.AnalogInputPort | undefined)[] => {
  const ctx = Form.useContext();
  const source = deep.widen(path);
  const cached = useRef<ChannelSource[]>([]);
  const getSnapshot = () => {
    const next = (ctx.subtrees?.() ?? [ctx]).map((c) => ({
      device: c.get(source.device, { optional: true })?.value ?? "",
      port: c.get(source.port, { optional: true })?.value,
    }));
    if (!deep.equal(next, cached.current)) cached.current = next;
    return cached.current;
  };
  const sources = useSyncExternalStore(ctx.bind, getSnapshot, getSnapshot);
  const devices = useByKeys(
    unique.unique(sources.map((s) => s.device)).filter(primitive.isNonZero),
  );
  return sources.map(({ device, port }) =>
    devices
      ?.find((d) => d.key === device)
      ?.properties.hardware?.analogInputs.find((p) => p.port === port),
  );
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
