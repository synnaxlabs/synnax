// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef } from "react";

import { Text, TextProps } from "./Text";

import "./TextLink.css";

export interface TextLinkProps extends Omit<TextProps, "ref"> {}

export const TextLink = forwardRef<HTMLAnchorElement, TextProps>(
  ({ href, download, target, rel, ...props }: TextLinkProps, ref): JSX.Element => (
    <a
      className="pluto-text-link"
      ref={ref}
      href={href}
      download={download}
      target={target}
      rel={rel}
    >
      <Text className="pluto-text-link__text" {...props}></Text>
    </a>
  )
);
TextLink.displayName = "TextLink";
