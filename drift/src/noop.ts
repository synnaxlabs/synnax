import { type Action,type UnknownAction } from "@reduxjs/toolkit";
import { type dimensions, type xy } from "@synnaxlabs/x";

import { type Event,type Runtime } from "@/runtime";
import { type StoreState } from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

/** 
 * In certain environments (such as the web browser), it is not really possible to
 * spawn new windows. The `NoopRuntime` is intended to stand in for drift in these 
 * environments.
 */
export class NoopRuntime<S extends StoreState, A extends Action = UnknownAction>
    implements Runtime<S, A> {

    async emit(_event: Omit<Event<S, A>, "emitter">, _to?: string): Promise<void> { }
    async subscribe(_lis: (event: Event<S, A>) => void): Promise<void> { }
    isMain(): boolean { return true; }
    label(): string { return MAIN_WINDOW; };
    onCloseRequested(_cb: () => void): void { };
    async listLabels(): Promise<string[]> { return []; }
    async getProps(): Promise<Omit<WindowProps, "key">> { return {}; }
    async create(_label: string, _props: Omit<WindowProps, "key">): Promise<void> { };
    async close(_label: string): Promise<void> { };
    async focus(): Promise<void> { };
    async setMinimized(_value: boolean): Promise<void> { };
    async setMaximized(_value: boolean): Promise<void> { };
    async setVisible(_value: boolean): Promise<void> { };
    async setFullscreen(_value: boolean): Promise<void> { };
    async center(): Promise<void> { };
    async setPosition(_xy: xy.XY): Promise<void> { };
    async setSize(_dims: dimensions.Dimensions): Promise<void> { };
    async setMinSize(_dims: dimensions.Dimensions): Promise<void> { };
    async setMaxSize(_dims: dimensions.Dimensions): Promise<void> { };
    async setResizable(_value: boolean): Promise<void> { };
    async setSkipTaskbar(_value: boolean): Promise<void> { };
    async setAlwaysOnTop(_value: boolean): Promise<void> { };
    async setDecorations(_value: boolean): Promise<void> { };
    async setTitle(_title: string): Promise<void> { };
    async configure(): Promise<void> { }
}
