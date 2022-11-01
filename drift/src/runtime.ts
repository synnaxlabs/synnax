import { Action, AnyAction } from "@reduxjs/toolkit";

import { PreloadedState, StoreState } from "./state";
import { KeyedWindowProps } from "./window";

/**
 * An event emitted by drift to communicate state changes.
 */
export type Event<S extends StoreState, A extends Action = AnyAction> = {
	/** The key of the window that emitted the event */
	emitter: string;
	/** A redux state action */
	action?: A;
	/** The entire redux store state. Sent only on the creation of new windows */
	state?: PreloadedState<S>;
	/** sendState is set to true when the window is requesting a state forward */
	sendState?: boolean;
};

export interface Sender<S extends StoreState, A extends Action = AnyAction> {
	/**
	 * Emits an event to all windows in the application.
	 * @param event - The event to emit.
	 * @param to - If set, the event will only be emitted to the window with the given key.
	 */
	emit(event: Omit<Event<S, A>, "emitter">, to?: string): void;
}

export interface Receiver<S extends StoreState, A extends Action = AnyAction> {
	/**
	 * Listens for an event from any window in the application.
	 * @param lis - The callback to call when the event is received.
	 */
	subscribe(lis: (event: Event<S, A>) => void): void;
}

export interface MainChecker {
	/**
	 * @returns true if the window is the main window of the application i.e. the first
	 * forked
	 */
	isMain(): boolean;
}

/**
 * Communicator allows for event communication between windows.
 */
export interface Communicator<S extends StoreState, A extends Action = AnyAction>
	extends Sender<S, A>,
		Receiver<S, A>,
		MainChecker {}

/**
 * Properties represents the runtime properties of a window.
 */
export interface Properties {
	/**
	 * @returns the key of the window.
	 */
	key(): string;
	/**
	 * Ready is called by drift when the current window has received state from the main
	 * window and is ready to be shown.
	 */
	ready(): void;
	/**
	 * Calls the provided function with the current window is closing.
	 */
	onCloseRequested(cb: () => void): void;
}

/**
 * Manager is used to manage the windows in the application.
 */
export interface Manager {
	/**
	 * Creates a new window with the given properties. The window should not be shown
	 * until the ready() method is called.
	 */
	create(props: KeyedWindowProps): void;
	/**
	 * Closes the window with the given key.
	 */
	close(key: string): void;
	/**
	 * Focuses the window with the given key.
	 */
	focus(key: string): void;
	/**
	 * Checks if the window with the given key exists.
	 */
	exists(key: string): boolean;
}

/**
 * An interface that represents the core runtime of the application.
 * Drift uses this runtime to manage windows and communicate between them.
 */
export interface Runtime<S extends StoreState, A extends Action = AnyAction>
	extends Communicator<S, A>,
		Properties,
		Manager {}
