class Canceled extends Error {
  static readonly MESSAGE = "canceled";
  constructor() {
    super(Canceled.MESSAGE);
  }

  /** Returns true if the error or message is a cancellation error" */
  matches(e: Error | string): boolean {
    if (typeof e === "string") return e.includes(Canceled.MESSAGE);
    return e instanceof Canceled || e.message.includes(Canceled.MESSAGE);
  }
}

/**
 * CANCELED should be thrown to indicate the cancellation of an operation.
 */
export const CANCELED = new Canceled();
