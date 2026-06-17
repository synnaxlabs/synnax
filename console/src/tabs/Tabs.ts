// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Drift } from "@synnaxlabs/drift";
import { type Icon, type Tabs as PTabs } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";

/** The placement location of a tab layout. */
export type PlacementLocation = "mosaic" | "window" | "modal";

/** Window construction properties for tab layouts opened in a standalone window. */
export type WindowProps = Omit<Drift.WindowProps, "key" | "url"> & {
  navTop?: boolean;
  showTitle?: boolean;
};

/** Properties used when a tab layout is placed within the mosaic. */
export interface TabProps extends Pick<PTabs.Tab, "closable" | "editable"> {
  tab?: PTabs.Tab;
  location?: location.Location;
  mosaicKey?: number;
}

/** A tab layout spec with an optional key; one is generated on placement if omitted. */
export interface BaseState<A = unknown> {
  key?: string;
  type: string;
  name: string;
  icon?: Icon.ReactElement | string;
  location?: PlacementLocation;
  window?: WindowProps;
  tab?: Partial<TabProps>;
  args?: A;
  excludeFromProject?: boolean;
  unsavedChanges?: boolean;
}

/** A fully specified, placed tab layout. */
export interface State<A = unknown> extends BaseState<A> {
  key: string;
  windowKey: string;
}
