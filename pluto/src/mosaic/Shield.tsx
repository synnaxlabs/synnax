// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/mosaic/Mosaic.css";

import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Haul } from "@/haul";
import { filterTabCreateHaulItems, filterTabDropHaulItems } from "@/mosaic/haul";

/**
 * Shield covers a leaf's content while a mosaic tab drag is in flight so drag events
 * reach the leaf instead of being swallowed by embedded content (canvases, iframes).
 * Render it inside the leaf's content region — a positioned container that does NOT
 * include the tab strip. Covering a dragged tab mid-dragstart cancels the native drag,
 * so the shield must never overlap the strip.
 */
export const Shield = (): ReactElement | null => {
  const items = Haul.useDraggingState().items;
  const active = useMemo(
    () =>
      filterTabDropHaulItems(items).length > 0 ||
      filterTabCreateHaulItems(items).length > 0,
    [items],
  );
  if (!active) return null;
  return <div className={CSS.BE("mosaic", "shield")} />;
};
