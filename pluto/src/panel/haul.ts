// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { z } from "zod";

const tabDragPayloadZ = z.object({ panel: panel.keyZ, tab: panel.tabZ });

/** The origin of a dragged tab: the panel holding it and the tab itself. */
export interface TabDragPayload extends z.infer<typeof tabDragPayloadZ> {}

/**
 * Builds the payload a dragged tab carries. The tab travels whole because the drop may
 * land in a window that has never loaded the source panel.
 */
export const createTabDragPayload = (
  key: panel.Key,
  tab: panel.Tab,
): record.Unknown => ({ panel: key, tab });

/** @returns the origin of a dragged tab, or undefined if data is not a tab payload. */
export const parseTabDragPayload = (data: unknown): TabDragPayload | undefined =>
  tabDragPayloadZ.safeParse(data).data;
