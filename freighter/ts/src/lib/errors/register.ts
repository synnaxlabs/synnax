import {FError, NONE, UNKNOWN} from "./errors";

type Payload = {
  type: string;
  data: string;
}

export class Register {
  private entries: { [type: string]: FError }

  constructor() {
    this.entries = {};
  }

  register(error: FError) {
    this.entries[error.type] = error;
  }

  encode(error: any): Payload {
    if (error === null || error === undefined) {
      return {type: NONE, data: ""};
    }
    if (error?.discriminator === 'FError') {
      return {
        type: error.type,
        data: error.encode()
      };
    }
    // If we can't parse the type, just try to stringify it.
    return {type: UNKNOWN, data: JSON.stringify(error)};
  }

  decode(payload: Payload): Error | null {
    if (payload.type === NONE) {
      return null;
    }
    if (this.entries[payload.type]) {
      return this.entries[payload.type].decode(payload.data);
    }
    // If we can't parse the type, just try to parse it.
    return Error(payload.data);
  }
}
