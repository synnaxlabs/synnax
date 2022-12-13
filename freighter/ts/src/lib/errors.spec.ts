import test from 'ava';

import {
  BaseTypedError,
  EOF,
  FREIGHTER,
  NONE,
  StreamClosed,
  TypedError,
  UNKNOWN,
  UnknownError,
  Unreachable,
  assertErrorType,
  decodeError,
  encodeError,
  isTypedError,
  registerError,
} from './errors';

class MyCustomError extends BaseTypedError {
  constructor(message: string) {
    super(message, 'MyCustomError');
  }
}

const myCustomErrorEncoder = (error: MyCustomError): string => {
  return error.message;
};

const myCustomErrorDecoder = (encoded: string): TypedError => {
  return new MyCustomError(encoded);
};

test('isTypedError', (t) => {
  const error = new MyCustomError('test');
  const fError = isTypedError(error);
  t.is(fError, true);
  t.is(error.type, 'MyCustomError');
});

test('encoding and decoding a custom error through registry', (t) => {
  registerError({
    type: 'MyCustomError',
    encode: myCustomErrorEncoder,
    decode: myCustomErrorDecoder,
  });
  const error = new MyCustomError('test');
  const encoded = encodeError(error);
  t.is(encoded.type, 'MyCustomError');
  t.is(encoded.data, 'test');
  const decoded = assertErrorType<MyCustomError>(
    'MyCustomError',
    decodeError(encoded)
  );
  t.is(decoded.message, 'test');
});

test('encoding and decoding a null error', (t) => {
  const encoded = encodeError(null);
  t.is(encoded.type, NONE);
  t.is(encoded.data, '');
  const decoded = decodeError(encoded);
  t.is(decoded, undefined);
});

test('encoding and decoding an unrecognized error', (t) => {
  const error = new Error('test');
  const encoded = encodeError(error);
  t.is(encoded.type, UNKNOWN);
  t.is(encoded.data, '{}');
  const decoded = decodeError(encoded);
  t.deepEqual(decoded, new UnknownError('{}'));
});

test('registering duplicate error should throw', (t) => {
  registerError({
    type: 'MyDuplicateError',
    encode: myCustomErrorEncoder,
    decode: myCustomErrorDecoder,
  });
  t.throws(() => {
    registerError({
      type: 'MyDuplicateError',
      encode: myCustomErrorEncoder,
      decode: myCustomErrorDecoder,
    });
  });
});

test('encoding and decoding freighter errors', (t) => {
  [new EOF(), new StreamClosed(), new Unreachable()].forEach((error) => {
    const encoded = encodeError(error);
    t.is(encoded.type, FREIGHTER);
    t.is(encoded.data, error.message);
    const decoded = decodeError(encoded);
    t.deepEqual(decoded, error);
  });
});
