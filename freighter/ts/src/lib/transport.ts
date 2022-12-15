import { Middleware } from "./middleware";

/**
 * Transport is a based interface that represents a general transport for
 * exchanging messages between a client and a server.
 */
export interface Transport {
  /**
   * Use registers middleware that will be executed in order when the transport
   *
   * @param mw - The middleware to register.
   */
  use(...mw: Middleware[]): void;
}
