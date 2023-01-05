// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Header as CoreHeader } from "./Header";
import { HeaderActions } from "./HeaderActions";
import { HeaderButtonTitle } from "./HeaderButtonTitle";
import { HeaderTitle } from "./HeaderTitle";

export type { HeaderProps } from "./Header";

type CoreHeaderType = typeof CoreHeader;

interface HeaderType extends CoreHeaderType {
  /**
   * Header.Button is a Header component whose text acts as a whose text field acts as a
   * text button component.
   *
   * @param props - The comonent props. The props of this component are identical to
   * those of Header, except for:
   * @param props.onClick - Called with a mouse event when the header button is clicked.
   */
  ButtonTitle: typeof HeaderButtonTitle;
  Title: typeof HeaderTitle;
  Actions: typeof HeaderActions;
}

/**
 * Header renders header text with a bottom border.
 *
 * @param props - The component props. All unused props will be passed down to the
 * {@link Space} containing the header.
 * @param props.level - The font level for the header. See the {@link Typography.Text}
 * component for all possible levels. Default is "h1."
 * @param props.icon - An optional icon to add add before the start of the header text.
 * @param props.actions - A list of actions to be rendered on the right side ofthe header.
 * If the action is a JSX element, it will be rendered directly, and if its of type
 * ButtonIconOnlyProps, a ButtonIconOnly will be rendered using the given props.
 * @param props.divided - If true, creates a divider between the start icon, header text,
 * and each action. Default is false.
 */
export const Header = CoreHeader as HeaderType;

Header.ButtonTitle = HeaderButtonTitle;
Header.Title = HeaderTitle;
Header.Actions = HeaderActions;
