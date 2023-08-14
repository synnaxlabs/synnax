import { Primitive, UnknownRecord } from "@synnaxlabs/x";

type State = Primitive | UnknownRecord;
export type SetStateFunc<S, PS = S> = (prev: PS) => S;

export const isStateSetter = <S extends State>(
  arg: SetStateArg<S>
): arg is SetStateFunc<S> => typeof arg === "function";
/** A function that mimics the behavior of a setState function from a usetState hook. */

export type SetStateArg<S extends State, PS = S> = S | SetStateFunc<S, PS>;
export type SetState<S extends State> = (value: SetStateArg<S>) => void;
export type InitialState<S extends State> = S | (() => S);

export const executeStateSetter = <S extends State>(
  setter: SetStateArg<S>,
  prev: S
): S => (isStateSetter(setter) ? setter(prev) : setter);

export const executeInitialSetter = <S extends State>(setter: InitialState<S>): S =>
  isInitialStateSetter(setter) ? setter() : setter;

export const isInitialStateSetter = <S extends State>(
  arg: InitialState<S>
): arg is () => S => typeof arg === "function";

export type UseState = <S extends State>(initial: InitialState<S>) => [S, SetState<S>];
export type PureUseState<S extends State> = () => [S, (value: S) => void];
