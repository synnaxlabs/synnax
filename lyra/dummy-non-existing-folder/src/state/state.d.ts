import { state } from "@synnaxlabs/x";
/** A state value and a setter that accepts a value or an updater. */
export type UseReturn<NextState extends state.State> = [
    NextState,
    state.Setter<NextState>
];
export type Use = <NextState extends state.State>(initial: state.Initial<NextState>) => UseReturn<NextState>;
/** A state value and a setter that accepts only a value, never an updater. */
export type PureUseReturn<NextState extends state.State> = [
    NextState,
    state.PureSetter<NextState>
];
export type PureUse<NextState extends state.State> = (initial: NextState) => PureUseReturn<NextState>;
/** Props for {@link usePassthrough}. */
export interface UsePassthroughProps<NextState extends state.State> {
    initial: state.Initial<NextState>;
    /** Set it, with `onChange`, to let the caller own the state. */
    value?: NextState;
    onChange?: state.Setter<NextState>;
}
/**
 * Lets a component be controlled or uncontrolled through one API: the caller owns the
 * state when it passes both `value` and `onChange`, and the component owns it
 * otherwise. `onChange` fires either way, so `value` decides who owns the state, not
 * who hears about it.
 */
export declare const usePassthrough: <NextState extends state.State>({ initial, value, onChange, }: UsePassthroughProps<NextState>) => UseReturn<NextState>;
/** Props for {@link usePurePassthrough}. */
export interface UsePurePassthroughProps<NextState extends state.State> {
    initialValue: state.Initial<NextState>;
    value?: NextState;
    onChange?: state.PureSetter<NextState>;
}
/** {@link usePassthrough} for a setter that takes only values, never updaters. */
export declare const usePurePassthrough: <NextState extends state.State>({ initialValue, value, onChange, }: UsePurePassthroughProps<NextState>) => PureUseReturn<NextState>;
/** State backed by local storage under the given key, restored on the next mount. */
export declare const usePersisted: <S extends state.State>(initial: state.Initial<S>, key: string) => UseReturn<S>;
//# sourceMappingURL=state.d.ts.map