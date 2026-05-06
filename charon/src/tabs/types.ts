// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode } from "react";
import { z } from "zod";

import { type RenderProp as BaseRenderProp } from "@/component/renderProp";
import { type Icon } from "@/icon";

export const specZ = z.object({
  tabKey: z.string(),
  name: z.string(),
  closable: z.boolean().optional(),
  icon: z.unknown().optional(),
  editable: z.boolean().optional(),
  visible: z.boolean().optional(),
  unsavedChanges: z.boolean().optional(),
  loading: z.boolean().optional(),
});

export interface Spec extends Omit<z.infer<typeof specZ>, "icon"> {
  icon?: Icon.ReactElement | string | unknown;
}
export const tabZ = specZ.extend({
  content: z.unknown().optional(),
});

export interface Tab extends Spec {
  content?: ReactNode | unknown;
}

export type RenderProp = BaseRenderProp<Tab>;
