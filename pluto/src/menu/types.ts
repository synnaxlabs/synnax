// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";

export const CONTEXT_SELECTED = CSS.BM("context", "selected");
export const CONTEXT_TARGET = CSS.BE("context", "target");
export const CONTEXT_MENU_CLASS = CSS.B("menu-context");

/**
 * Attribute stamped on the context target a menu opened over, for as long as the
 * menu is visible, letting CSS hold the target's hover styling. An attribute
 * rather than a class: React rewrites className on re-render, which would wipe an
 * imperative class.
 */
export const CONTEXT_OPEN_ATTRIBUTE = "data-context-menu-open";
