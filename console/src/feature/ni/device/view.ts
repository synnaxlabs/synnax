// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device, ni, Synnax } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, primitive } from "@synnaxlabs/x";

import { useByKeys, useTypedByKeys } from "@/feature/ni/device/queries";
import { type Device, SCHEMAS } from "@/feature/ni/device/types";
import { FLAGS } from "@/flags";

/** The fields of an NI device that task forms read, from either device table. */
export interface View extends Pick<
  ni.device.Device,
  "rack" | "name" | "model" | "identifier" | "hardware"
> {
  key: device.Key;
  /** The DAQmx alias of the device. Empty for a typed device, which stores none. */
  location: string;
}

/** The view of a device from the generic device table. */
export const toView = ({
  key,
  rack,
  name,
  model,
  location,
  properties: { identifier, hardware },
}: Device): View => ({ key, rack, name, model, location, identifier, hardware });

const useGenericViews = (keys: device.Key[]): View[] | undefined =>
  useByKeys(keys)?.map(toView);

/** The view of a device from the typed device table. */
export const fromTyped = ({
  key,
  rack,
  name,
  model,
  identifier,
  hardware,
}: ni.device.Device): View => ({
  key,
  rack,
  name,
  model,
  location: "",
  identifier,
  hardware,
});

const useTypedViews = (keys: device.Key[]): View[] | undefined =>
  useTypedByKeys(keys)?.map(fromTyped);

const useViews = FLAGS.typedNIDevices ? useTypedViews : useGenericViews;

const retrieveGenericViews = async (
  client: Synnax,
  keys: device.Key[],
): Promise<View[]> =>
  (await client.devices.retrieve({ key: keys, schemas: SCHEMAS })).map(toView);

const retrieveTypedViews = async (
  client: Synnax,
  keys: device.Key[],
): Promise<View[]> => (await client.ni.devices.retrieve(keys)).map(fromTyped);

/**
 * Retrieves the views of the devices the keys name. Reads the typed device table when
 * `typedNIDevices` is on.
 */
export const retrieveViews = FLAGS.typedNIDevices
  ? retrieveTypedViews
  : retrieveGenericViews;

const NO_KEYS: device.Key[] = [];

/** The view of the device with the key, or undefined for an empty key. */
export const useView = (key: device.Key): View | undefined =>
  useViews(primitive.isNonZero(key) ? [key] : NO_KEYS)?.[0];

const path = deep.path<{ config: { device: device.Key } }>();

/** The key of the device the form's config names, or no keys when it names none. */
export const useConfigDeviceKeys = (): device.Key[] => {
  const key = Form.useFieldValue(path.config.device, { optional: true }) ?? "";
  return primitive.isNonZero(key) ? [key] : NO_KEYS;
};

/** The view of the device the form's config names. */
export const useViewFromConfig = (): View | undefined =>
  useViews(useConfigDeviceKeys())?.[0];
