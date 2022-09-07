import test from 'ava';

import {
  assertErrorType,
  BaseTypedError,
  decodeError,
  encodeError,
  isTypedError,
  registerError,
  TypedError,
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

test('encoding an decoding through registry', (t) => {
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
