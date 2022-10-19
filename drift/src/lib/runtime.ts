import { PayloadAction } from '@reduxjs/toolkit';

export type WindowProps = Omit<KeyedWindowProps, 'key'> & {
  key?: string;
};

export type KeyedWindowProps = {
  key: string;
  url?: string;
  center?: boolean;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  minWidth?: number;
  minHeight?: number;
  maxWidth?: number;
  maxHeight?: number;
  resizable?: boolean;
  title?: string;
  fullscreen?: boolean;
  focus?: boolean;
  transparent?: boolean;
  maximized?: boolean;
  visible?: boolean;
  decorations?: boolean;
  skipTaskbar?: boolean;
  fileDropEnabled?: boolean;
};

export interface Runtime {
  release(): void;
  isMain(): boolean;
  winKey(): string;
  createWindow(props: KeyedWindowProps): void;
  ready(): void;
  emit(event: Event): void;
  subscribe(lis: (event: Event) => void): void;
}

export type Event = {
  winKey: string;
  action?: PayloadAction<unknown>;
  state?: any;
  sendInitialState?: boolean;
};
