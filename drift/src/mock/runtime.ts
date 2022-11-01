import { StoreState } from "@/state";
import { Event, Runtime } from "@/runtime";
import { Action, AnyAction } from "@reduxjs/toolkit";
import { KeyedWindowProps } from "@/window";

export class MockRuntime<S extends StoreState, A extends Action = AnyAction>
	implements Runtime<S, A>
{
	_isMain: boolean = false;
	_key: string = "mock";
	markedReady: boolean = false;
	emissions: Event<S, A>[] = [];
	hasCreated: KeyedWindowProps[] = [];
	hasClosed: string[] = [];
	hasFocused: string[] = [];
	subscribeCallback: (event: Event<S, A>) => void = () => {};
	requestClosure: () => void = () => {};

	constructor(isMain: boolean) {
		this._isMain = isMain;
	}

	isMain(): boolean {
		return this._isMain;
	}

	key(): string {
		return this._key;
	}

	emit(event: Omit<Event<S, A>, "emitter">, to?: string): void {
		this.emissions.push({ ...event, emitter: this.key() });
	}

	subscribe(lis: (event: Event<S, A>) => void): void {
		this.subscribeCallback = lis;
	}

	ready() {
		this.markedReady = true;
	}

	create(props: KeyedWindowProps) {
		this.hasCreated.push(props);
	}

	close(key: string) {
		this.hasClosed.push(key);
	}

	onCloseRequested(cb: () => void): void {
		this.requestClosure = cb;
	}

	focus(key: string) {
		this.hasFocused.push(key);
	}

	exists(key: string): boolean {
		// check if in list of created and NOT in list of closed
		const hasBeenCreated = this.hasCreated.some((w) => w.key === key);
		const hasBeenClosed = this.hasClosed.some((w) => w === key);
		return hasBeenCreated && !hasBeenClosed;
	}
}
