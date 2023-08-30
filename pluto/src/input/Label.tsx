// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HTMLAttributes, ReactElement } from "react";

import { CSS } from "@/css";

import "@/input/Label.css";

/** Props for the {@link Label} component. */
export interface LabelProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLLabelElement>, HTMLLabelElement> {}

/**
 * A thin, styled wrapper for an input label. We generally recommend using Input.Item
 * with a 'label' prop instead of this component. This component is useful for
 * low-level control over the label element.
 *
 * @param props - Props for the label component. Unlisted props are passed to the
 * underlying label element.
 */
export const Label = ({ className, ...props }: LabelProps): ReactElement => {
  return <label className={CSS(CSS.B("input-label"), className)} {...props} />;
};
