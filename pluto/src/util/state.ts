import { Primitive, UnknownRecord } from "@synnaxlabs/x";

type State = Primitive | UnknownRecord;
export type SetStateFunc<S, PS = S> = (prev: PS) => S;

export const isStateSetter = <S extends State>(
  arg: SetStateArg<S>
): arg is SetStateFunc<S> => typeof arg === "function";
/** A function that mimics the behavior of a setState function from a usetState hook. */

export type SetStateArg<S extends State, PS = S> = S | SetStateFunc<S, PS>;
export type SetState<S extends State> = (value: SetStateArg<S>) => void;
export type InitialState<S extends Primitive | object> = S | (() => S);

export const executeStateSetter = <S extends State>(
  setter: SetStateArg<S>,
  prev: S
): S => (isStateSetter(setter) ? setter(prev) : setter);
