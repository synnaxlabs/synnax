// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TextLinkProps } from "../Typography";

import { Button, ButtonProps } from "./Button";

/** Props for the {@link ButtonLink} component. */
export interface ButtonLinkProps
  extends ButtonProps,
    Pick<TextLinkProps, "href" | "target"> {}

export const ButtonLink = ({
  href,
  target,
  results,
  ...props
}: ButtonLinkProps): JSX.Element => {
  return (
    <form action={href} target={target} rel={target}>
      <Button {...props} />
    </form>
  );
};
