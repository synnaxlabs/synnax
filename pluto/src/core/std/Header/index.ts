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
import { HeaderTitle as CoreHeaderTitle } from "./HeaderTitle";

export type { HeaderProps } from "./Header";
export type { HeaderTitleProps } from "./HeaderTitle";
export type { HeaderButtonTitleProps } from "./HeaderButtonTitle";
export type { HeaderActionsProps } from "./HeaderActions";

type CoreHeaderTitleType = typeof CoreHeaderTitle;

export interface HeaderTitleType extends CoreHeaderTitleType {
  /**
   * Header.Title.Button renders a clickable header title.
   *
   * @param props - The comonent props. The props for this component are identical
   * to {@link Button}, except the variant is always 'outlined' and that the component size
   * is automatically inferred from the 'level' prop passsed to the parent {@link Header}
   * component.
   */
  Button: typeof HeaderButtonTitle;
}

export const HeaderTitle = CoreHeaderTitle as HeaderTitleType;

HeaderTitle.Button = HeaderButtonTitle;

type CoreHeaderType = typeof CoreHeader;

interface HeaderType extends CoreHeaderType {
  /**
   * Renders the title for the header component.
   *
   * @param props - The component props. The props for this component are identical
   * to the {@link Typography.TextWithIcon} component, except that the 'level', and
   * 'divider' props are inherited from the parent {@link Header} component.
   */
  Title: typeof HeaderTitle;
  /**
   * Custom actions to render on the right side of the header.
   *
   * @param children - The actions to render. If the action is of type
   * {@link ButtonIconProps}, a correectly sized {@link ButtonIconOnly} is rendered
   * using the given props. If the action is a JSX element, it is renderered directly.
   * It's a good idea to prefer the latter in almost all cases for simplicity.
   */
  Actions: typeof HeaderActions;
}

/**
 * Header is the container for a m
 *
 * @param props - The component props. All unused props will be passed down to the
 * {@link Space} containing the header.
 * @param props.level - The font level for the header. See the {@link Typography.Text}
 * component for all possible levels. Default is "h1."
 * @param props.icon - An optional icon to add add before the start of the header text.
 * @param props.actions - A list of actions to be rendered on the right side ofthe header.
 * If the action is a JSX element, it will be rendered directly, and if its of type
 * ButtonIconProps, a ButtonIconOnly will be rendered using the given props.
 * @param props.divided - If true, creates a divider between the start icon, header text,
 * and each action. Default is false.
 */
export const Header = CoreHeader as HeaderType;

Header.Title = HeaderTitle;
Header.Actions = HeaderActions;
