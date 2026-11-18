// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SlaveOnRack } from "@/feature/ethercat/device/types";

/** A new slave, and the old slaves that it can take the place of. */
export interface Replacement {
  /** A connected slave that the user has not configured. */
  slave: SlaveOnRack;
  /**
   * The configured, enabled, and disconnected slaves of its rack, network, vendor, and
   * product, in order of position.
   */
  old: SlaveOnRack[];
}

const sameModel = (a: SlaveOnRack, b: SlaveOnRack): boolean =>
  a.rack === b.rack &&
  a.properties.network === b.properties.network &&
  a.properties.vendorId === b.properties.vendorId &&
  a.properties.productCode === b.properties.productCode;

/**
 * Returns whether the new slave can take the place of the old slave: the new slave is
 * connected and not configured, and the old slave is configured, enabled, and
 * disconnected, on the same rack and network, with the same vendor and product. The
 * revision does not count, since a slave on its other bus port gives another revision.
 */
export const canReplace = (old: SlaveOnRack, slave: SlaveOnRack): boolean =>
  slave.connected &&
  !slave.configured &&
  old.configured &&
  old.properties.enabled &&
  !old.connected &&
  sameModel(old, slave);

/**
 * Returns each new slave in slaves that can take the place of an old slave, with the
 * old slaves it can replace.
 */
export const findReplacements = (slaves: SlaveOnRack[]): Replacement[] =>
  slaves
    .map((slave) => ({
      slave,
      old: slaves
        .filter((o) => canReplace(o, slave))
        .sort((a, b) => a.properties.position - b.properties.position),
    }))
    .filter(({ old }) => old.length > 0);
