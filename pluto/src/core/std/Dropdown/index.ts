// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dropdown as CoreDropdown, useDropdown } from "@/core/std/Dropdown/Dropdown";
export type { DropdownProps, UseDropdownReturn } from "@/core/std/Dropdown/Dropdown";

type CoreDropdownType = typeof CoreDropdown;

export interface DropdownType extends CoreDropdownType {
  /**
   * Implements basic dropdown behavior, and should be preferred when using
   * the {@link Dropdown} component. Opens the dropdown whenever the 'open' function is
   * called, and closes it whenever the 'close' function is called OR the user clicks
   * outside of the dropdown parent wrapped,which includes the dropdown trigger (often
   * a button or input).
   *
   * @param initialVisible - Whether the dropdown should be visible on mount.
   * @returns visible - Whether the dropdown is visible.
   * @returns ref - The ref to the dropdown parent.
   * @returns close - A function to close the dropdown.
   * @returns open - A function to open the dropdown.
   * @returns toggle - A function to toggle the dropdown.
   */
  use: typeof useDropdown;
}

/**
 * A controlled dropdown component that wraps its parent. For the simplest case, use
 * the {@link useDropdown} hook (more behavioral details explained there).
 *
 * @param props - The props for the dropdown component. Unlisted props are passed to the
 * parent elment.
 * @param props.visible - Whether the dropdown is visible or not. This is a controlled
 * @param props.children - Two children are expected: the dropdown trigger (often a button
 * or input) and the dropdown content.
 */
export const Dropdown = CoreDropdown as DropdownType;

Dropdown.use = useDropdown;
