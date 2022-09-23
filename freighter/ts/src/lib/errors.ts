/**
 * @description FError is an interface for an error that can be transported over
 * the network.
 */
import { z } from 'zod';

export interface TypedError extends Error {
  discriminator: 'FreighterError';
  /**
   * @description Returns a unique type identifier for the error. Freighter uses this to
   * determine the correct decoder to use on the other end of the freighter.
   */
  type: string;
}

export class BaseTypedError extends Error implements TypedError {
  discriminator: 'FreighterError' = 'FreighterError';
  type: string;

  constructor(message: string, type: string) {
    super(message);
    this.type = type;
  }
}

type ErrorDecoder = (encoded: string) => Error | undefined;
type ErrorEncoder = (error: TypedError) => string;

export const isTypedError = (error: unknown): error is TypedError => {
  if (!error || typeof error !== 'object') {
    return false;
  }
  const typedError = error as TypedError;
  if (typedError.discriminator !== 'FreighterError') {
    return false;
  }
  if (!('type' in typedError)) {
    throw new Error(
      `Freighter error is missing its type property: ${typedError}`
    );
  }
  return true;
};

export const assertErrorType = <T>(type: string, error?: Error): T => {
  if (!error) {
    throw new Error(`Expected error of type ${type} but got nothing instead`);
  }
  if (!isTypedError(error)) {
    throw new Error(`Expected a typed error, got: ${error}`);
  }
  if (error.type !== type) {
    throw new Error(
      `Expected error of type ${type}, got ${error.type}: ${error}`
    );
  }
  return error as unknown as T;
};

export const UNKNOWN = 'unknown';
export const NONE = 'nil';
export const FREIGHTER = 'freighter';

export const ErrorPayloadSchema = z.object({
  type: z.string(),
  data: z.string(),
});

export type ErrorPayload = z.infer<typeof ErrorPayloadSchema>;

type errorProvider = {
  encode: ErrorEncoder;
  decode: ErrorDecoder;
};

class Registry {
  private entries: { [type: string]: errorProvider };

  constructor() {
    this.entries = {};
  }

  register(_type: string, provider: errorProvider) {
    if (this.entries[_type]) {
      throw new Error(`Error type ${_type} is already registered`);
    }
    this.entries[_type] = provider;
  }

  encode(error: unknown): ErrorPayload {
    if (!error) {
      return { type: NONE, data: '' };
    }
    if (isTypedError(error) && this.entries[error.type]) {
      return { type: error.type, data: this.entries[error.type].encode(error) };
    }
    return { type: UNKNOWN, data: JSON.stringify(error) };
  }

  decode(payload: ErrorPayload): Error | undefined {
    if (payload.type === NONE) {
      return undefined;
    }

    if (payload.type === UNKNOWN) {
      return new UnknownError(payload.data);
    }

    const provider = this.entries[payload.type];
    if (!provider) {
      return new UnknownError(payload.data);
    }
    return provider.decode(payload.data);
  }
}

const REGISTRY = new Registry();

export const registerError = (props: {
  type: string;
  encode: ErrorEncoder;
  decode: ErrorDecoder;
}) => {
  const { type, ...provider } = props;
  REGISTRY.register(type, provider);
};

export const encodeError = (error: unknown): ErrorPayload => {
  return REGISTRY.encode(error);
};

export const decodeError = (payload: ErrorPayload): Error | undefined => {
  return REGISTRY.decode(payload);
};

export class UnknownError extends BaseTypedError implements TypedError {
  constructor(message: string) {
    super(message, UNKNOWN);
  }
}

export class EOF extends BaseTypedError implements TypedError {
  constructor() {
    super('EOF', FREIGHTER);
  }
}

export class StreamClosed extends BaseTypedError implements TypedError {
  constructor() {
    super('StreamClosed', FREIGHTER);
  }
}

export class Unreachable extends BaseTypedError implements TypedError {
  constructor() {
    super('Unreachable', FREIGHTER);
  }
}

const freighterErrorEncoder: ErrorEncoder = (error: TypedError) => {
  if (error instanceof EOF) {
    return 'EOF';
  }
  if (error instanceof StreamClosed) {
    return 'StreamClosed';
  }
  if (error instanceof Unreachable) {
    return 'Unreachable';
  }
  throw new Error(`Unknown error type: ${error}`);
};

const freighterErrorDecoder: ErrorDecoder = (encoded: string) => {
  switch (encoded) {
    case 'EOF':
      return new EOF();
    case 'StreamClosed':
      return new StreamClosed();
    case 'Unreachable':
      return new Unreachable();
    default:
      throw new Error(`Unknown error type: ${encoded}`);
  }
};

registerError({
  type: FREIGHTER,
  encode: freighterErrorEncoder,
  decode: freighterErrorDecoder,
});
