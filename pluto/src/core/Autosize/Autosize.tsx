// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent, ReactElement, cloneElement, useRef } from "react";

import { useSize } from "@/hooks";

/* AutoSize props is the props for the {@link AutoSize} component. */
export interface AutosizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  children: FunctionComponent<{ width: number; height: number }> | ReactElement;
  debounce?: number;
}

/**
 * AutoSize renders and tracks the dimensions of a div element and provides them to its
 * children.
 *
 * @param props - Standard div props that can be used to apply styles, classnames, etc.
 * to the parent div being measured.
 * @param props.debounce - An optional debounce time to set the maximum rate at which
 * AutoSize will rerender its children with updated dimensions. (A debounce time of
 * 100ms will mean that only one resize event will be propagated even if the component
 * is resized twice).
 */
export const Autosize = ({
  children,
  debounce,
  ...props
}: AutosizeProps): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);
  const size = useSize({ ref, debounce });
  const content: ReactElement | null =
    typeof children === "function" ? children(size) : cloneElement(children, size);
  return (
    <div ref={ref} {...props}>
      {content}
    </div>
  );
};
