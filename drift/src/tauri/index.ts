import { Action, AnyAction } from "@reduxjs/toolkit";
import { Event as TauriEvent, UnlistenFn, emit, listen } from "@tauri-apps/api/event";
import { WebviewWindow, appWindow } from "@tauri-apps/api/window";

import { Event, Runtime } from "@/runtime";
import { StoreState } from "@/state";
import { KeyedWindowProps, MAIN_WINDOW } from "@/window";

const actionEvent = "action";
const tauriError = "tauri://error";
const notFound = (key: string) => new Error(`Window not found: ${key}`);

const encode = <S extends StoreState, A extends Action = AnyAction>(
	event: Event<S, A>
) => JSON.stringify(event);
const decode = <S extends StoreState, A extends Action = AnyAction>(event: string) =>
	JSON.parse(event) as Event<S, A>;

/**
 * A Tauri backed implementation of the drift Runtime.
 */
export class TauriRuntime<S extends StoreState, A extends Action = AnyAction>
	implements Runtime<S, A>
{
	private window: WebviewWindow;
	private unsubscribe?: void | UnlistenFn;

	/**
	 * @param window - The WebviewWindow to use as the underlying engine for this runtime.
	 * This should not be set in 99% of cases. Only use this if you know what you're doing.
	 */
	constructor(window?: WebviewWindow) {
		this.window = window || appWindow;
	}

	key(): string {
		return this.window.label;
	}

	isMain(): boolean {
		return this.window.label === MAIN_WINDOW;
	}

	release() {
		this.unsubscribe && this.unsubscribe();
	}

	ready(): void {
		this.window.show();
	}

	create({ key, ...props }: KeyedWindowProps) {
		const w = new WebviewWindow(key as string, {
			...props,
			visible: false,
		});
		w.once(tauriError, console.error);
	}

	emit(event_: Omit<Event<S, A>, "emitter">, to?: string): void {
		const event = encode({ ...event_, emitter: this.key() });
		if (to) {
			const win = WebviewWindow.getByLabel(to);
			if (!win) throw notFound(to);
			win.emit(actionEvent, event);
		} else {
			emit(actionEvent, event);
		}
	}

	subscribe(lis: (action: Event<S, A>) => void): void {
		listen<string>(actionEvent, (event: TauriEvent<string>) =>
			lis(decode(event.payload))
		)
			.catch(console.error)
			.then((unlisten) => {
				this.unsubscribe = unlisten;
			});
	}

	onCloseRequested(cb: () => void): void {
		this.window.onCloseRequested((e) => {
			// Only propagate the close request if the event
			// is for the current window.
			if (e.windowLabel === this.key()) {
				// Prevent default so the window doesn't close
				// until all processes are complete.
				e.preventDefault();
				cb();
			}
		});
	}

	close(key: string): void {
		const win = WebviewWindow.getByLabel(key);
		if (win) win.close();
	}

	focus(key: string): void {
		const win = WebviewWindow.getByLabel(key);
		if (win) win.setFocus();
	}

	exists(key: string): boolean {
		return !!WebviewWindow.getByLabel(key);
	}
}
