/**
 * @description FError is an interface for an error that can be transported over
 * a freighter freighter.
 */
export interface FError {
  discriminator: 'FError';
  /**
   * @description Returns a unique type identifier for the error. Freighter uses this to
   * determine the correct decoder to use on the other end of the freighter.
   */
  type: string;

  /**
   * @description Encodes the error into a string for freighter.
   */
  encode(): string;

  /**
   * @description Decodes the error from a string.
   */
  decode(encoded: string): Error;
}

export const UNKNOWN = "unknown";
export const NONE = "nil";
