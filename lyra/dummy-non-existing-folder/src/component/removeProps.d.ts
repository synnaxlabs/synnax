import { type FC } from "react";
/**
 * A higher-order component that removes specified props from a wrapped component. This
 * is useful when you want to prevent certain props from being passed down to a child
 * component.
 * @param {FC<P>} WrappedComponent - The component to wrap
 * @param {string[]} propsToRemove - Array of prop names to remove from the wrapped
 * component
 * @returns {FC<P>} A new component that filters out the specified props before passing
 * them to the wrapped component
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
export declare const removeProps: <P extends object>(WrappedComponent: FC<P>, propsToRemove: string[]) => FC<P>;
//# sourceMappingURL=removeProps.d.ts.map