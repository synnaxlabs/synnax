import test from 'ava';

import Endpoint from './endpoint';

test('[endpoint] - test path', (t) => {
  const endpoint = new Endpoint({
    host: 'localhost',
    port: 8080,
    protocol: 'http',
    pathPrefix: 'api',
  });
  t.is(endpoint.path('test'), 'http://localhost:8080/api/test');
});

test('[endpoint] - child', (t) => {
  const endpoint = new Endpoint({
    host: 'localhost',
    port: 8080,
    protocol: 'http',
    pathPrefix: 'api',
  });
  const child = endpoint.child({ path: 'test' });
  t.is(child.path('test'), 'http://localhost:8080/api/test/test');
});
