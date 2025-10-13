// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/toolbar/Header.css";

import { Button, Flex, Header as PHeader } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface ContentProps extends Flex.BoxProps {}

export const Content = (props: ContentProps): ReactElement => (
  <Flex.Box empty y pack full {...props} />
);

export interface HeaderProps extends PHeader.HeaderProps {
  padded?: boolean;
}

export const Header = ({ padded, ...rest }: HeaderProps): ReactElement => (
  <PHeader.Header
    className={CSS(
      CSS.BE("toolbar", "header"),
      padded && CSS.BEM("toolbar", "header", "padded"),
    )}
    level="h5"
    shrink={false}
    background={1}
    {...rest}
  />
);

export const Actions = (props: PHeader.ActionsProps): ReactElement => (
  <PHeader.Actions {...props} />
);

export interface ActionProps extends Button.ButtonProps {}

export const Action = ({ className, ...rest }: ActionProps): ReactElement => (
  <Button.Button
    contrast={2}
    size="small"
    rounded={1}
    className={CSS(CSS.BE("toolbar", "action"), className)}
    {...rest}
  />
);
