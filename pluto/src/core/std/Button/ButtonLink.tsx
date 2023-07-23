// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Button, ButtonProps } from "@/core/std/Button";
import { TextLinkProps } from "@/core/std/Typography";

/** Props for the {@link ButtonLink} component. */
export interface ButtonLinkProps
  extends ButtonProps,
    Pick<TextLinkProps, "href" | "target"> {}

export const ButtonLink = ({
  href,
  target,
  ...props
}: ButtonLinkProps): ReactElement => {
  return (
    <form action={href} target={target} rel={target}>
      <Button {...props} />
    </form>
  );
};
