// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";

/**
 * A higher-order component that removes specified props from a wrapped component.
 * This is useful when you want to prevent certain props from being passed down
 * to a child component.
 *
 * @template P - The type of props accepted by the wrapped component
 *
 * @param {FC<P>} WrappedComponent - The component to wrap
 * @param {string[]} propsToRemove - Array of prop names to remove from the wrapped component
 *
 * @returns {FC<P>} A new component that filters out the specified props
 * before passing them to the wrapped component
 *
 * @example
 * // Remove the 'className' prop from a Button component
 * const ButtonWithoutClassName = removeProps(Button, ['className']);
 *
 * // Usage
 * <ButtonWithoutClassName
 *   className="will-be-removed"
 *   onClick={() => {}} // other props pass through normally
 * />
 */
export const removeProps = <P extends object>(
  WrappedComponent: FC<P>,
  propsToRemove: string[],
): FC<P> => {
  const omitted: Record<string, never> = {};
  propsToRemove.forEach((prop) => (omitted[prop] = undefined as never));
  const C = (props: P) => {
    const { ...restProps } = { ...props, ...omitted };
    return <WrappedComponent {...(restProps as P)} />;
  };
  C.displayName = `RemoveProps(${WrappedComponent.displayName})`;
  return C;
};
