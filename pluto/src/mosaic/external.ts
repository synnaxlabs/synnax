// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export * from "@/mosaic/Mosaic";
export * from "@/mosaic/tree";
export * from "@/mosaic/types";
export * from "@/mosaic/use";
// The legacy tab-serialization model still backs the mosaic layout until the mosaic
// redesign lands. Surfaced here so consumers reach it through the Mosaic namespace.
export { DefaultName } from "@/tabs/legacy/Selector";
export {
  type NameProps,
  type RenderProp,
  type Spec,
  type Tab,
  tabZ,
} from "@/tabs/legacy/types";
