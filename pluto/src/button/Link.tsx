// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { type Text } from "@/text";

/** Props for the {@link Link} component. */
export interface LinkProps<L extends Text.Level = "h1">
  extends ButtonProps,
    Pick<Text.LinkProps<L>, "href" | "target"> {}
/**
 * Use.Link renders a button that looks like a link and redirects to the given href
 * when clicked.
 *
 * @param props - Props for the component. Identical to the props for the Use component,
 * excluding 'variant', and  adding the following:
 * @param props.href - The URL to redirect to when the button is clicked.
 * @param props.target - The target of the link. Defaults to "_self".
 */

export const Link = <L extends Text.Level = "h1">({
  href,
  target,
  ...props
}: LinkProps<L>): ReactElement => {
  // @ts-expect-error - generic element issues
  return <Button<"a"> el="a" href={href} {...props} />;
};
