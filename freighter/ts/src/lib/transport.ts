import { Middleware } from './middleware';

export interface Transport {
  use(...mw: Middleware[]): void;
}
