import _n, { createContext as Tr, useContext as yn, useState as vn, useEffect as ee, useMemo as Sr, cloneElement as Cr, useRef as Ar, createElement as kr } from "react";
const st = (...t) => t.join(" ");
var he = { exports: {} }, ct = {};
/**
 * @license React
 * react-jsx-runtime.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
var Xe;
function Nr() {
  if (Xe)
    return ct;
  Xe = 1;
  var t = _n, e = Symbol.for("react.element"), n = Symbol.for("react.fragment"), r = Object.prototype.hasOwnProperty, i = t.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner, o = { key: !0, ref: !0, __self: !0, __source: !0 };
  function s(u, l, f) {
    var c, p = {}, g = null, _ = null;
    f !== void 0 && (g = "" + f), l.key !== void 0 && (g = "" + l.key), l.ref !== void 0 && (_ = l.ref);
    for (c in l)
      r.call(l, c) && !o.hasOwnProperty(c) && (p[c] = l[c]);
    if (u && u.defaultProps)
      for (c in l = u.defaultProps, l)
        p[c] === void 0 && (p[c] = l[c]);
    return { $$typeof: e, type: u, key: g, ref: _, props: p, _owner: i.current };
  }
  return ct.Fragment = n, ct.jsx = s, ct.jsxs = s, ct;
}
var ft = {};
/**
 * @license React
 * react-jsx-runtime.development.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
var Ue;
function $r() {
  return Ue || (Ue = 1, process.env.NODE_ENV !== "production" && function() {
    var t = _n, e = Symbol.for("react.element"), n = Symbol.for("react.portal"), r = Symbol.for("react.fragment"), i = Symbol.for("react.strict_mode"), o = Symbol.for("react.profiler"), s = Symbol.for("react.provider"), u = Symbol.for("react.context"), l = Symbol.for("react.forward_ref"), f = Symbol.for("react.suspense"), c = Symbol.for("react.suspense_list"), p = Symbol.for("react.memo"), g = Symbol.for("react.lazy"), _ = Symbol.for("react.offscreen"), E = Symbol.iterator, C = "@@iterator";
    function F(a) {
      if (a === null || typeof a != "object")
        return null;
      var h = E && a[E] || a[C];
      return typeof h == "function" ? h : null;
    }
    var D = t.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED;
    function R(a) {
      {
        for (var h = arguments.length, d = new Array(h > 1 ? h - 1 : 0), y = 1; y < h; y++)
          d[y - 1] = arguments[y];
        H("error", a, d);
      }
    }
    function H(a, h, d) {
      {
        var y = D.ReactDebugCurrentFrame, x = y.getStackAddendum();
        x !== "" && (h += "%s", d = d.concat([x]));
        var b = d.map(function(m) {
          return String(m);
        });
        b.unshift("Warning: " + h), Function.prototype.apply.call(console[a], console, b);
      }
    }
    var j = !1, xt = !1, Gn = !1, Jn = !1, Kn = !1, xe;
    xe = Symbol.for("react.module.reference");
    function Zn(a) {
      return !!(typeof a == "string" || typeof a == "function" || a === r || a === o || Kn || a === i || a === f || a === c || Jn || a === _ || j || xt || Gn || typeof a == "object" && a !== null && (a.$$typeof === g || a.$$typeof === p || a.$$typeof === s || a.$$typeof === u || a.$$typeof === l || a.$$typeof === xe || a.getModuleId !== void 0));
    }
    function Qn(a, h, d) {
      var y = a.displayName;
      if (y)
        return y;
      var x = h.displayName || h.name || "";
      return x !== "" ? d + "(" + x + ")" : d;
    }
    function be(a) {
      return a.displayName || "Context";
    }
    function q(a) {
      if (a == null)
        return null;
      if (typeof a.tag == "number" && R("Received an unexpected object in getComponentNameFromType(). This is likely a bug in React. Please file an issue."), typeof a == "function")
        return a.displayName || a.name || null;
      if (typeof a == "string")
        return a;
      switch (a) {
        case r:
          return "Fragment";
        case n:
          return "Portal";
        case o:
          return "Profiler";
        case i:
          return "StrictMode";
        case f:
          return "Suspense";
        case c:
          return "SuspenseList";
      }
      if (typeof a == "object")
        switch (a.$$typeof) {
          case u:
            var h = a;
            return be(h) + ".Consumer";
          case s:
            var d = a;
            return be(d._context) + ".Provider";
          case l:
            return Qn(a, a.render, "ForwardRef");
          case p:
            var y = a.displayName || null;
            return y !== null ? y : q(a.type) || "Memo";
          case g: {
            var x = a, b = x._payload, m = x._init;
            try {
              return q(m(b));
            } catch {
              return null;
            }
          }
        }
      return null;
    }
    var G = Object.assign, ut = 0, Ee, Re, Te, Se, Ce, Ae, ke;
    function Ne() {
    }
    Ne.__reactDisabledLog = !0;
    function tr() {
      {
        if (ut === 0) {
          Ee = console.log, Re = console.info, Te = console.warn, Se = console.error, Ce = console.group, Ae = console.groupCollapsed, ke = console.groupEnd;
          var a = {
            configurable: !0,
            enumerable: !0,
            value: Ne,
            writable: !0
          };
          Object.defineProperties(console, {
            info: a,
            log: a,
            warn: a,
            error: a,
            group: a,
            groupCollapsed: a,
            groupEnd: a
          });
        }
        ut++;
      }
    }
    function er() {
      {
        if (ut--, ut === 0) {
          var a = {
            configurable: !0,
            enumerable: !0,
            writable: !0
          };
          Object.defineProperties(console, {
            log: G({}, a, {
              value: Ee
            }),
            info: G({}, a, {
              value: Re
            }),
            warn: G({}, a, {
              value: Te
            }),
            error: G({}, a, {
              value: Se
            }),
            group: G({}, a, {
              value: Ce
            }),
            groupCollapsed: G({}, a, {
              value: Ae
            }),
            groupEnd: G({}, a, {
              value: ke
            })
          });
        }
        ut < 0 && R("disabledDepth fell below zero. This is a bug in React. Please file an issue.");
      }
    }
    var Bt = D.ReactCurrentDispatcher, Xt;
    function bt(a, h, d) {
      {
        if (Xt === void 0)
          try {
            throw Error();
          } catch (x) {
            var y = x.stack.trim().match(/\n( *(at )?)/);
            Xt = y && y[1] || "";
          }
        return `
` + Xt + a;
      }
    }
    var Ut = !1, Et;
    {
      var nr = typeof WeakMap == "function" ? WeakMap : Map;
      Et = new nr();
    }
    function $e(a, h) {
      if (!a || Ut)
        return "";
      {
        var d = Et.get(a);
        if (d !== void 0)
          return d;
      }
      var y;
      Ut = !0;
      var x = Error.prepareStackTrace;
      Error.prepareStackTrace = void 0;
      var b;
      b = Bt.current, Bt.current = null, tr();
      try {
        if (h) {
          var m = function() {
            throw Error();
          };
          if (Object.defineProperty(m.prototype, "props", {
            set: function() {
              throw Error();
            }
          }), typeof Reflect == "object" && Reflect.construct) {
            try {
              Reflect.construct(m, []);
            } catch (B) {
              y = B;
            }
            Reflect.construct(a, [], m);
          } else {
            try {
              m.call();
            } catch (B) {
              y = B;
            }
            a.call(m.prototype);
          }
        } else {
          try {
            throw Error();
          } catch (B) {
            y = B;
          }
          a();
        }
      } catch (B) {
        if (B && y && typeof B.stack == "string") {
          for (var w = B.stack.split(`
`), N = y.stack.split(`
`), T = w.length - 1, S = N.length - 1; T >= 1 && S >= 0 && w[T] !== N[S]; )
            S--;
          for (; T >= 1 && S >= 0; T--, S--)
            if (w[T] !== N[S]) {
              if (T !== 1 || S !== 1)
                do
                  if (T--, S--, S < 0 || w[T] !== N[S]) {
                    var M = `
` + w[T].replace(" at new ", " at ");
                    return a.displayName && M.includes("<anonymous>") && (M = M.replace("<anonymous>", a.displayName)), typeof a == "function" && Et.set(a, M), M;
                  }
                while (T >= 1 && S >= 0);
              break;
            }
        }
      } finally {
        Ut = !1, Bt.current = b, er(), Error.prepareStackTrace = x;
      }
      var et = a ? a.displayName || a.name : "", Be = et ? bt(et) : "";
      return typeof a == "function" && Et.set(a, Be), Be;
    }
    function rr(a, h, d) {
      return $e(a, !1);
    }
    function ir(a) {
      var h = a.prototype;
      return !!(h && h.isReactComponent);
    }
    function Rt(a, h, d) {
      if (a == null)
        return "";
      if (typeof a == "function")
        return $e(a, ir(a));
      if (typeof a == "string")
        return bt(a);
      switch (a) {
        case f:
          return bt("Suspense");
        case c:
          return bt("SuspenseList");
      }
      if (typeof a == "object")
        switch (a.$$typeof) {
          case l:
            return rr(a.render);
          case p:
            return Rt(a.type, h, d);
          case g: {
            var y = a, x = y._payload, b = y._init;
            try {
              return Rt(b(x), h, d);
            } catch {
            }
          }
        }
      return "";
    }
    var Tt = Object.prototype.hasOwnProperty, Pe = {}, Oe = D.ReactDebugCurrentFrame;
    function St(a) {
      if (a) {
        var h = a._owner, d = Rt(a.type, a._source, h ? h.type : null);
        Oe.setExtraStackFrame(d);
      } else
        Oe.setExtraStackFrame(null);
    }
    function or(a, h, d, y, x) {
      {
        var b = Function.call.bind(Tt);
        for (var m in a)
          if (b(a, m)) {
            var w = void 0;
            try {
              if (typeof a[m] != "function") {
                var N = Error((y || "React class") + ": " + d + " type `" + m + "` is invalid; it must be a function, usually from the `prop-types` package, but received `" + typeof a[m] + "`.This often happens because of typos such as `PropTypes.function` instead of `PropTypes.func`.");
                throw N.name = "Invariant Violation", N;
              }
              w = a[m](h, m, y, d, null, "SECRET_DO_NOT_PASS_THIS_OR_YOU_WILL_BE_FIRED");
            } catch (T) {
              w = T;
            }
            w && !(w instanceof Error) && (St(x), R("%s: type specification of %s `%s` is invalid; the type checker function must return `null` or an `Error` but returned a %s. You may have forgotten to pass an argument to the type checker creator (arrayOf, instanceOf, objectOf, oneOf, oneOfType, and shape all require an argument).", y || "React class", d, m, typeof w), St(null)), w instanceof Error && !(w.message in Pe) && (Pe[w.message] = !0, St(x), R("Failed %s type: %s", d, w.message), St(null));
          }
      }
    }
    var ar = Array.isArray;
    function Wt(a) {
      return ar(a);
    }
    function sr(a) {
      {
        var h = typeof Symbol == "function" && Symbol.toStringTag, d = h && a[Symbol.toStringTag] || a.constructor.name || "Object";
        return d;
      }
    }
    function ur(a) {
      try {
        return Fe(a), !1;
      } catch {
        return !0;
      }
    }
    function Fe(a) {
      return "" + a;
    }
    function Me(a) {
      if (ur(a))
        return R("The provided key is an unsupported type %s. This value must be coerced to a string before before using it here.", sr(a)), Fe(a);
    }
    var lt = D.ReactCurrentOwner, lr = {
      key: !0,
      ref: !0,
      __self: !0,
      __source: !0
    }, De, Ie, jt;
    jt = {};
    function cr(a) {
      if (Tt.call(a, "ref")) {
        var h = Object.getOwnPropertyDescriptor(a, "ref").get;
        if (h && h.isReactWarning)
          return !1;
      }
      return a.ref !== void 0;
    }
    function fr(a) {
      if (Tt.call(a, "key")) {
        var h = Object.getOwnPropertyDescriptor(a, "key").get;
        if (h && h.isReactWarning)
          return !1;
      }
      return a.key !== void 0;
    }
    function hr(a, h) {
      if (typeof a.ref == "string" && lt.current && h && lt.current.stateNode !== h) {
        var d = q(lt.current.type);
        jt[d] || (R('Component "%s" contains the string ref "%s". Support for string refs will be removed in a future major release. This case cannot be automatically converted to an arrow function. We ask you to manually fix this case by using useRef() or createRef() instead. Learn more about using refs safely here: https://reactjs.org/link/strict-mode-string-ref', q(lt.current.type), a.ref), jt[d] = !0);
      }
    }
    function pr(a, h) {
      {
        var d = function() {
          De || (De = !0, R("%s: `key` is not a prop. Trying to access it will result in `undefined` being returned. If you need to access the same value within the child component, you should pass it as a different prop. (https://reactjs.org/link/special-props)", h));
        };
        d.isReactWarning = !0, Object.defineProperty(a, "key", {
          get: d,
          configurable: !0
        });
      }
    }
    function dr(a, h) {
      {
        var d = function() {
          Ie || (Ie = !0, R("%s: `ref` is not a prop. Trying to access it will result in `undefined` being returned. If you need to access the same value within the child component, you should pass it as a different prop. (https://reactjs.org/link/special-props)", h));
        };
        d.isReactWarning = !0, Object.defineProperty(a, "ref", {
          get: d,
          configurable: !0
        });
      }
    }
    var gr = function(a, h, d, y, x, b, m) {
      var w = {
        $$typeof: e,
        type: a,
        key: h,
        ref: d,
        props: m,
        _owner: b
      };
      return w._store = {}, Object.defineProperty(w._store, "validated", {
        configurable: !1,
        enumerable: !1,
        writable: !0,
        value: !1
      }), Object.defineProperty(w, "_self", {
        configurable: !1,
        enumerable: !1,
        writable: !1,
        value: y
      }), Object.defineProperty(w, "_source", {
        configurable: !1,
        enumerable: !1,
        writable: !1,
        value: x
      }), Object.freeze && (Object.freeze(w.props), Object.freeze(w)), w;
    };
    function _r(a, h, d, y, x) {
      {
        var b, m = {}, w = null, N = null;
        d !== void 0 && (Me(d), w = "" + d), fr(h) && (Me(h.key), w = "" + h.key), cr(h) && (N = h.ref, hr(h, x));
        for (b in h)
          Tt.call(h, b) && !lr.hasOwnProperty(b) && (m[b] = h[b]);
        if (a && a.defaultProps) {
          var T = a.defaultProps;
          for (b in T)
            m[b] === void 0 && (m[b] = T[b]);
        }
        if (w || N) {
          var S = typeof a == "function" ? a.displayName || a.name || "Unknown" : a;
          w && pr(m, S), N && dr(m, S);
        }
        return gr(a, w, N, x, y, lt.current, m);
      }
    }
    var Gt = D.ReactCurrentOwner, ze = D.ReactDebugCurrentFrame;
    function tt(a) {
      if (a) {
        var h = a._owner, d = Rt(a.type, a._source, h ? h.type : null);
        ze.setExtraStackFrame(d);
      } else
        ze.setExtraStackFrame(null);
    }
    var Jt;
    Jt = !1;
    function Kt(a) {
      return typeof a == "object" && a !== null && a.$$typeof === e;
    }
    function He() {
      {
        if (Gt.current) {
          var a = q(Gt.current.type);
          if (a)
            return `

Check the render method of \`` + a + "`.";
        }
        return "";
      }
    }
    function yr(a) {
      {
        if (a !== void 0) {
          var h = a.fileName.replace(/^.*[\\\/]/, ""), d = a.lineNumber;
          return `

Check your code at ` + h + ":" + d + ".";
        }
        return "";
      }
    }
    var Ve = {};
    function vr(a) {
      {
        var h = He();
        if (!h) {
          var d = typeof a == "string" ? a : a.displayName || a.name;
          d && (h = `

Check the top-level render call using <` + d + ">.");
        }
        return h;
      }
    }
    function Le(a, h) {
      {
        if (!a._store || a._store.validated || a.key != null)
          return;
        a._store.validated = !0;
        var d = vr(h);
        if (Ve[d])
          return;
        Ve[d] = !0;
        var y = "";
        a && a._owner && a._owner !== Gt.current && (y = " It was passed a child from " + q(a._owner.type) + "."), tt(a), R('Each child in a list should have a unique "key" prop.%s%s See https://reactjs.org/link/warning-keys for more information.', d, y), tt(null);
      }
    }
    function Ye(a, h) {
      {
        if (typeof a != "object")
          return;
        if (Wt(a))
          for (var d = 0; d < a.length; d++) {
            var y = a[d];
            Kt(y) && Le(y, h);
          }
        else if (Kt(a))
          a._store && (a._store.validated = !0);
        else if (a) {
          var x = F(a);
          if (typeof x == "function" && x !== a.entries)
            for (var b = x.call(a), m; !(m = b.next()).done; )
              Kt(m.value) && Le(m.value, h);
        }
      }
    }
    function wr(a) {
      {
        var h = a.type;
        if (h == null || typeof h == "string")
          return;
        var d;
        if (typeof h == "function")
          d = h.propTypes;
        else if (typeof h == "object" && (h.$$typeof === l || h.$$typeof === p))
          d = h.propTypes;
        else
          return;
        if (d) {
          var y = q(h);
          or(d, a.props, "prop", y, a);
        } else if (h.PropTypes !== void 0 && !Jt) {
          Jt = !0;
          var x = q(h);
          R("Component %s declared `PropTypes` instead of `propTypes`. Did you misspell the property assignment?", x || "Unknown");
        }
        typeof h.getDefaultProps == "function" && !h.getDefaultProps.isReactClassApproved && R("getDefaultProps is only used on classic React.createClass definitions. Use a static property named `defaultProps` instead.");
      }
    }
    function mr(a) {
      {
        for (var h = Object.keys(a.props), d = 0; d < h.length; d++) {
          var y = h[d];
          if (y !== "children" && y !== "key") {
            tt(a), R("Invalid prop `%s` supplied to `React.Fragment`. React.Fragment can only have `key` and `children` props.", y), tt(null);
            break;
          }
        }
        a.ref !== null && (tt(a), R("Invalid attribute `ref` supplied to `React.Fragment`."), tt(null));
      }
    }
    function qe(a, h, d, y, x, b) {
      {
        var m = Zn(a);
        if (!m) {
          var w = "";
          (a === void 0 || typeof a == "object" && a !== null && Object.keys(a).length === 0) && (w += " You likely forgot to export your component from the file it's defined in, or you might have mixed up default and named imports.");
          var N = yr(x);
          N ? w += N : w += He();
          var T;
          a === null ? T = "null" : Wt(a) ? T = "array" : a !== void 0 && a.$$typeof === e ? (T = "<" + (q(a.type) || "Unknown") + " />", w = " Did you accidentally export a JSX literal instead of a component?") : T = typeof a, R("React.jsx: type is invalid -- expected a string (for built-in components) or a class/function (for composite components) but got: %s.%s", T, w);
        }
        var S = _r(a, h, d, x, b);
        if (S == null)
          return S;
        if (m) {
          var M = h.children;
          if (M !== void 0)
            if (y)
              if (Wt(M)) {
                for (var et = 0; et < M.length; et++)
                  Ye(M[et], a);
                Object.freeze && Object.freeze(M);
              } else
                R("React.jsx: Static children should always be an array. You are likely explicitly calling React.jsxs or React.jsxDEV. Use the Babel transform instead.");
            else
              Ye(M, a);
        }
        return a === r ? mr(S) : wr(S), S;
      }
    }
    function xr(a, h, d) {
      return qe(a, h, d, !0);
    }
    function br(a, h, d) {
      return qe(a, h, d, !1);
    }
    var Er = br, Rr = xr;
    ft.Fragment = r, ft.jsx = Er, ft.jsxs = Rr;
  }()), ft;
}
(function(t) {
  process.env.NODE_ENV === "production" ? t.exports = Nr() : t.exports = $r();
})(he);
const A = he.exports.jsx, Vt = he.exports.jsxs;
function ms({
  children: t,
  size: e = "medium",
  variant: n = "filled",
  className: r,
  ...i
}) {
  return /* @__PURE__ */ A("button", {
    className: st("pluto-btn", "pluto-btn--" + n, "pluto-btn--" + e, r),
    ...i,
    children: t
  });
}
const Pr = {
  start: "flex-start",
  end: "flex-end",
  center: "center",
  spaceBetween: "space-between",
  spaceAround: "space-around",
  spaceEvenly: "space-evenly"
}, pe = ({
  empty: t = !1,
  size: e = "medium",
  justify: n = "start",
  children: r,
  align: i,
  ...o
}) => {
  let s;
  return t ? s = 0 : typeof e == "string" ? s = `pluto-space--${e}` : s = `calc(var(--base-size) * ${e})`, /* @__PURE__ */ A("div", {
    className: st("pluto-space", typeof e == "string" ? "pluto-space--" + e : void 0, o.className),
    style: {
      flexDirection: o.direction === "horizontal" ? "row" : "column",
      gap: s,
      justifyContent: n && Pr[n],
      alignItems: i,
      ...o.style
    },
    children: r
  });
};
var Ct, Or = new Uint8Array(16);
function Fr() {
  if (!Ct && (Ct = typeof crypto < "u" && crypto.getRandomValues && crypto.getRandomValues.bind(crypto) || typeof msCrypto < "u" && typeof msCrypto.getRandomValues == "function" && msCrypto.getRandomValues.bind(msCrypto), !Ct))
    throw new Error("crypto.getRandomValues() not supported. See https://github.com/uuidjs/uuid#getrandomvalues-not-supported");
  return Ct(Or);
}
const Mr = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;
function Dr(t) {
  return typeof t == "string" && Mr.test(t);
}
var k = [];
for (var Zt = 0; Zt < 256; ++Zt)
  k.push((Zt + 256).toString(16).substr(1));
function Ir(t) {
  var e = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : 0, n = (k[t[e + 0]] + k[t[e + 1]] + k[t[e + 2]] + k[t[e + 3]] + "-" + k[t[e + 4]] + k[t[e + 5]] + "-" + k[t[e + 6]] + k[t[e + 7]] + "-" + k[t[e + 8]] + k[t[e + 9]] + "-" + k[t[e + 10]] + k[t[e + 11]] + k[t[e + 12]] + k[t[e + 13]] + k[t[e + 14]] + k[t[e + 15]]).toLowerCase();
  if (!Dr(n))
    throw TypeError("Stringified UUID is invalid");
  return n;
}
function zr(t, e, n) {
  t = t || {};
  var r = t.random || (t.rng || Fr)();
  if (r[6] = r[6] & 15 | 64, r[8] = r[8] & 63 | 128, e) {
    n = n || 0;
    for (var i = 0; i < 16; ++i)
      e[n + i] = r[i];
    return e;
  }
  return Ir(r);
}
const Hr = 6, Vr = () => zr().substring(0, Hr), xs = ({
  size: t = "medium",
  name: e = Vr(),
  label: n,
  placeholder: r,
  value: i,
  ...o
}) => /* @__PURE__ */ Vt(pe, {
  className: "pluto-input__container",
  size: 1,
  children: [n && /* @__PURE__ */ A("label", {
    className: "pluto-input__label",
    htmlFor: e,
    children: n
  }), /* @__PURE__ */ A("input", {
    id: e,
    placeholder: r,
    className: st("pluto-input__input", "pluto-input__input--" + t, o.className),
    ...o
  })]
});
const We = "#FFFFFF", je = "#212429", Lr = "Roboto, sans-serif", $ = 6, V = {
  name: "arya-base",
  colors: {
    primary: {
      m1: "#3363BE",
      z: "#3774D0",
      p1: "#3b84e5"
    },
    gray: {
      p2: "#51565e",
      p1: "#61636b",
      z: "#ACB5BD",
      m1: "#b2b2b2",
      m2: "#c9c9c9"
    },
    error: {
      m1: "#CF1322",
      z: "#F5222D",
      p1: "#FF4547"
    },
    visualization: {
      palettes: {
        default: [
          "#DC136C",
          "#20A4F3",
          "#7AC74F",
          "#FFC43D",
          "#FE5F55",
          "#8075FF",
          "#470063",
          "#020877",
          "#D90429"
        ]
      }
    },
    white: We,
    black: je,
    background: We,
    text: je
  },
  sizes: {
    base: $,
    border: {
      radius: 2,
      width: 1
    }
  },
  typography: {
    family: Lr,
    h1: {
      size: $ * 7,
      weight: "500",
      lineHeight: $ * 8
    },
    h2: {
      size: $ * 6,
      weight: "medium",
      lineHeight: $ * 7
    },
    h3: {
      size: $ * 4,
      weight: "medium",
      lineHeight: $ * 5
    },
    h4: {
      size: $ * 3.5,
      weight: "medium",
      lineHeight: $ * 4
    },
    h5: {
      size: $ * 2,
      weight: "medium",
      lineHeight: $ * 3,
      textTransform: "uppercase"
    },
    p: {
      size: $ * 2.5,
      weight: "regular",
      lineHeight: $ * 3.5
    },
    small: {
      size: $ * 2,
      weight: "regular",
      lineHeight: $ * 3
    }
  }
}, Yr = {
  ...V,
  name: "arya-light"
}, bs = {
  ...V,
  name: "arya-dark",
  colors: {
    ...V.colors,
    gray: {
      m2: V.colors.gray.p2,
      m1: V.colors.gray.p1,
      z: V.colors.gray.z,
      p1: V.colors.gray.m1,
      p2: V.colors.gray.m2
    },
    background: V.colors.black,
    text: V.colors.white
  }
}, qr = (t, e) => {
  v(t, "--primary-m1", e.colors.primary.m1), v(t, "--primary-z", e.colors.primary.z), v(t, "--primary-p1", e.colors.primary.p1), v(t, "--gray-m2", e.colors.gray.m2), v(t, "--gray-m1", e.colors.gray.m1), v(t, "--gray-z", e.colors.gray.z), v(t, "--gray-p1", e.colors.gray.p1), v(t, "--gray-p2", e.colors.gray.p2), v(t, "--error-m1", e.colors.error.m1), v(t, "--error-z", e.colors.error.z), v(t, "--error-p1", e.colors.error.p1), v(t, "--white", e.colors.white), v(t, "--black", e.colors.black), v(t, "--background", e.colors.background), v(t, "--text-color", e.colors.text), v(t, "--base-size", e.sizes.base), v(t, "--border-radius", e.sizes.border.radius), v(t, "--border-width", e.sizes.border.width), v(t, "--font-family", e.typography.family), v(t, "--h1-font-size", e.typography.h1.size), v(t, "--h1-line-height", e.typography.h1.lineHeight), v(t, "--h1-weight", e.typography.h1.weight), v(t, "--h2-font-size", e.typography.h2.size), v(t, "--h2-line-height", e.typography.h2.lineHeight), v(t, "--h2-weight", e.typography.h2.weight), v(t, "--h3-font-size", e.typography.h3.size), v(t, "--h3-line-height", e.typography.h3.lineHeight), v(t, "--h3-weight", e.typography.h3.weight), v(t, "--h4-font-size", e.typography.h4.size), v(t, "--h4-line-height", e.typography.h4.lineHeight), v(t, "--h4-weight", e.typography.h4.weight), v(t, "--h5-font-size", e.typography.h5.size), v(t, "--h5-line-height", e.typography.h5.lineHeight), v(t, "--h5-weight", e.typography.h5.weight), v(
    t,
    "--h5-text-transform",
    e.typography.h2.textTransform
  ), v(t, "--p-font-size", e.typography.p.size), v(t, "--p-line-height", e.typography.p.lineHeight), v(t, "--p-weight", e.typography.p.weight), v(t, "--small-font-size", e.typography.small.size), v(
    t,
    "--small-line-height",
    e.typography.small.lineHeight
  ), v(t, "--small-weight", e.typography.small.weight);
}, v = (t, e, n) => {
  n != null && (typeof n == "number" && (n = n + "px"), t.style.setProperty(e, String(n)));
};
const Br = ({
  className: t,
  ...e
}) => /* @__PURE__ */ Vt("label", {
  className: st("pluto-switch__container", t),
  children: [/* @__PURE__ */ A("input", {
    className: "pluto-switch__input",
    type: "checkbox",
    ...e
  }), /* @__PURE__ */ A("span", {
    className: "pluto-switch__slider"
  })]
}), de = Tr({
  theme: Yr,
  toggleTheme: () => {
    console.log("unimp");
  }
}), wn = () => yn(de), Es = ({
  children: t,
  themes: e
}) => {
  const [n, r] = vn(0), i = () => {
    console.log("toggleTheme"), n === e.length - 1 ? r(0) : r(n + 1);
  };
  ee(() => {
    qr(document.documentElement, e[n]);
  }, [n]);
  const o = Sr(() => e[n], [n]);
  return console.log(i), /* @__PURE__ */ A(de.Provider, {
    value: {
      theme: o,
      toggleTheme: i
    },
    children: t
  });
}, Rs = () => {
  const {
    theme: t,
    toggleTheme: e
  } = yn(de);
  return /* @__PURE__ */ A(Br, {
    onChange: () => {
      e();
    }
  });
}, Xr = (t, e) => ({
  h1: /* @__PURE__ */ A("h1", {
    children: e
  }),
  h2: /* @__PURE__ */ A("h2", {
    children: e
  }),
  h3: /* @__PURE__ */ A("h3", {
    children: e
  }),
  h4: /* @__PURE__ */ A("h4", {
    children: e
  }),
  h5: /* @__PURE__ */ A("h5", {
    children: e
  }),
  p: /* @__PURE__ */ A("p", {
    children: e
  }),
  small: /* @__PURE__ */ A("h6", {
    children: e
  })
})[t], Ts = ({
  size: t,
  text: e,
  icon: n,
  style: r,
  className: i,
  ...o
}) => {
  const {
    theme: s
  } = wn(), u = s.typography[t].size;
  return /* @__PURE__ */ Vt(pe, {
    direction: "horizontal",
    className: st("pluto-header__container", i),
    align: "center",
    size: "medium",
    style: {
      height: u,
      ...r
    },
    ...o,
    children: [n && Cr(n, {
      size: u
    }), Xr(t, e)]
  });
};
var Ur = { value: () => {
} };
function mn() {
  for (var t = 0, e = arguments.length, n = {}, r; t < e; ++t) {
    if (!(r = arguments[t] + "") || r in n || /[\s.]/.test(r))
      throw new Error("illegal type: " + r);
    n[r] = [];
  }
  return new $t(n);
}
function $t(t) {
  this._ = t;
}
function Wr(t, e) {
  return t.trim().split(/^|\s+/).map(function(n) {
    var r = "", i = n.indexOf(".");
    if (i >= 0 && (r = n.slice(i + 1), n = n.slice(0, i)), n && !e.hasOwnProperty(n))
      throw new Error("unknown type: " + n);
    return { type: n, name: r };
  });
}
$t.prototype = mn.prototype = {
  constructor: $t,
  on: function(t, e) {
    var n = this._, r = Wr(t + "", n), i, o = -1, s = r.length;
    if (arguments.length < 2) {
      for (; ++o < s; )
        if ((i = (t = r[o]).type) && (i = jr(n[i], t.name)))
          return i;
      return;
    }
    if (e != null && typeof e != "function")
      throw new Error("invalid callback: " + e);
    for (; ++o < s; )
      if (i = (t = r[o]).type)
        n[i] = Ge(n[i], t.name, e);
      else if (e == null)
        for (i in n)
          n[i] = Ge(n[i], t.name, null);
    return this;
  },
  copy: function() {
    var t = {}, e = this._;
    for (var n in e)
      t[n] = e[n].slice();
    return new $t(t);
  },
  call: function(t, e) {
    if ((i = arguments.length - 2) > 0)
      for (var n = new Array(i), r = 0, i, o; r < i; ++r)
        n[r] = arguments[r + 2];
    if (!this._.hasOwnProperty(t))
      throw new Error("unknown type: " + t);
    for (o = this._[t], r = 0, i = o.length; r < i; ++r)
      o[r].value.apply(e, n);
  },
  apply: function(t, e, n) {
    if (!this._.hasOwnProperty(t))
      throw new Error("unknown type: " + t);
    for (var r = this._[t], i = 0, o = r.length; i < o; ++i)
      r[i].value.apply(e, n);
  }
};
function jr(t, e) {
  for (var n = 0, r = t.length, i; n < r; ++n)
    if ((i = t[n]).name === e)
      return i.value;
}
function Ge(t, e, n) {
  for (var r = 0, i = t.length; r < i; ++r)
    if (t[r].name === e) {
      t[r] = Ur, t = t.slice(0, r).concat(t.slice(r + 1));
      break;
    }
  return n != null && t.push({ name: e, value: n }), t;
}
var ne = "http://www.w3.org/1999/xhtml";
const Je = {
  svg: "http://www.w3.org/2000/svg",
  xhtml: ne,
  xlink: "http://www.w3.org/1999/xlink",
  xml: "http://www.w3.org/XML/1998/namespace",
  xmlns: "http://www.w3.org/2000/xmlns/"
};
function Lt(t) {
  var e = t += "", n = e.indexOf(":");
  return n >= 0 && (e = t.slice(0, n)) !== "xmlns" && (t = t.slice(n + 1)), Je.hasOwnProperty(e) ? { space: Je[e], local: t } : t;
}
function Gr(t) {
  return function() {
    var e = this.ownerDocument, n = this.namespaceURI;
    return n === ne && e.documentElement.namespaceURI === ne ? e.createElement(t) : e.createElementNS(n, t);
  };
}
function Jr(t) {
  return function() {
    return this.ownerDocument.createElementNS(t.space, t.local);
  };
}
function xn(t) {
  var e = Lt(t);
  return (e.local ? Jr : Gr)(e);
}
function Kr() {
}
function ge(t) {
  return t == null ? Kr : function() {
    return this.querySelector(t);
  };
}
function Zr(t) {
  typeof t != "function" && (t = ge(t));
  for (var e = this._groups, n = e.length, r = new Array(n), i = 0; i < n; ++i)
    for (var o = e[i], s = o.length, u = r[i] = new Array(s), l, f, c = 0; c < s; ++c)
      (l = o[c]) && (f = t.call(l, l.__data__, c, o)) && ("__data__" in l && (f.__data__ = l.__data__), u[c] = f);
  return new O(r, this._parents);
}
function Qr(t) {
  return t == null ? [] : Array.isArray(t) ? t : Array.from(t);
}
function ti() {
  return [];
}
function bn(t) {
  return t == null ? ti : function() {
    return this.querySelectorAll(t);
  };
}
function ei(t) {
  return function() {
    return Qr(t.apply(this, arguments));
  };
}
function ni(t) {
  typeof t == "function" ? t = ei(t) : t = bn(t);
  for (var e = this._groups, n = e.length, r = [], i = [], o = 0; o < n; ++o)
    for (var s = e[o], u = s.length, l, f = 0; f < u; ++f)
      (l = s[f]) && (r.push(t.call(l, l.__data__, f, s)), i.push(l));
  return new O(r, i);
}
function En(t) {
  return function() {
    return this.matches(t);
  };
}
function Rn(t) {
  return function(e) {
    return e.matches(t);
  };
}
var ri = Array.prototype.find;
function ii(t) {
  return function() {
    return ri.call(this.children, t);
  };
}
function oi() {
  return this.firstElementChild;
}
function ai(t) {
  return this.select(t == null ? oi : ii(typeof t == "function" ? t : Rn(t)));
}
var si = Array.prototype.filter;
function ui() {
  return Array.from(this.children);
}
function li(t) {
  return function() {
    return si.call(this.children, t);
  };
}
function ci(t) {
  return this.selectAll(t == null ? ui : li(typeof t == "function" ? t : Rn(t)));
}
function fi(t) {
  typeof t != "function" && (t = En(t));
  for (var e = this._groups, n = e.length, r = new Array(n), i = 0; i < n; ++i)
    for (var o = e[i], s = o.length, u = r[i] = [], l, f = 0; f < s; ++f)
      (l = o[f]) && t.call(l, l.__data__, f, o) && u.push(l);
  return new O(r, this._parents);
}
function Tn(t) {
  return new Array(t.length);
}
function hi() {
  return new O(this._enter || this._groups.map(Tn), this._parents);
}
function Ft(t, e) {
  this.ownerDocument = t.ownerDocument, this.namespaceURI = t.namespaceURI, this._next = null, this._parent = t, this.__data__ = e;
}
Ft.prototype = {
  constructor: Ft,
  appendChild: function(t) {
    return this._parent.insertBefore(t, this._next);
  },
  insertBefore: function(t, e) {
    return this._parent.insertBefore(t, e);
  },
  querySelector: function(t) {
    return this._parent.querySelector(t);
  },
  querySelectorAll: function(t) {
    return this._parent.querySelectorAll(t);
  }
};
function pi(t) {
  return function() {
    return t;
  };
}
function di(t, e, n, r, i, o) {
  for (var s = 0, u, l = e.length, f = o.length; s < f; ++s)
    (u = e[s]) ? (u.__data__ = o[s], r[s] = u) : n[s] = new Ft(t, o[s]);
  for (; s < l; ++s)
    (u = e[s]) && (i[s] = u);
}
function gi(t, e, n, r, i, o, s) {
  var u, l, f = /* @__PURE__ */ new Map(), c = e.length, p = o.length, g = new Array(c), _;
  for (u = 0; u < c; ++u)
    (l = e[u]) && (g[u] = _ = s.call(l, l.__data__, u, e) + "", f.has(_) ? i[u] = l : f.set(_, l));
  for (u = 0; u < p; ++u)
    _ = s.call(t, o[u], u, o) + "", (l = f.get(_)) ? (r[u] = l, l.__data__ = o[u], f.delete(_)) : n[u] = new Ft(t, o[u]);
  for (u = 0; u < c; ++u)
    (l = e[u]) && f.get(g[u]) === l && (i[u] = l);
}
function _i(t) {
  return t.__data__;
}
function yi(t, e) {
  if (!arguments.length)
    return Array.from(this, _i);
  var n = e ? gi : di, r = this._parents, i = this._groups;
  typeof t != "function" && (t = pi(t));
  for (var o = i.length, s = new Array(o), u = new Array(o), l = new Array(o), f = 0; f < o; ++f) {
    var c = r[f], p = i[f], g = p.length, _ = vi(t.call(c, c && c.__data__, f, r)), E = _.length, C = u[f] = new Array(E), F = s[f] = new Array(E), D = l[f] = new Array(g);
    n(c, p, C, F, D, _, e);
    for (var R = 0, H = 0, j, xt; R < E; ++R)
      if (j = C[R]) {
        for (R >= H && (H = R + 1); !(xt = F[H]) && ++H < E; )
          ;
        j._next = xt || null;
      }
  }
  return s = new O(s, r), s._enter = u, s._exit = l, s;
}
function vi(t) {
  return typeof t == "object" && "length" in t ? t : Array.from(t);
}
function wi() {
  return new O(this._exit || this._groups.map(Tn), this._parents);
}
function mi(t, e, n) {
  var r = this.enter(), i = this, o = this.exit();
  return typeof t == "function" ? (r = t(r), r && (r = r.selection())) : r = r.append(t + ""), e != null && (i = e(i), i && (i = i.selection())), n == null ? o.remove() : n(o), r && i ? r.merge(i).order() : i;
}
function xi(t) {
  for (var e = t.selection ? t.selection() : t, n = this._groups, r = e._groups, i = n.length, o = r.length, s = Math.min(i, o), u = new Array(i), l = 0; l < s; ++l)
    for (var f = n[l], c = r[l], p = f.length, g = u[l] = new Array(p), _, E = 0; E < p; ++E)
      (_ = f[E] || c[E]) && (g[E] = _);
  for (; l < i; ++l)
    u[l] = n[l];
  return new O(u, this._parents);
}
function bi() {
  for (var t = this._groups, e = -1, n = t.length; ++e < n; )
    for (var r = t[e], i = r.length - 1, o = r[i], s; --i >= 0; )
      (s = r[i]) && (o && s.compareDocumentPosition(o) ^ 4 && o.parentNode.insertBefore(s, o), o = s);
  return this;
}
function Ei(t) {
  t || (t = Ri);
  function e(p, g) {
    return p && g ? t(p.__data__, g.__data__) : !p - !g;
  }
  for (var n = this._groups, r = n.length, i = new Array(r), o = 0; o < r; ++o) {
    for (var s = n[o], u = s.length, l = i[o] = new Array(u), f, c = 0; c < u; ++c)
      (f = s[c]) && (l[c] = f);
    l.sort(e);
  }
  return new O(i, this._parents).order();
}
function Ri(t, e) {
  return t < e ? -1 : t > e ? 1 : t >= e ? 0 : NaN;
}
function Ti() {
  var t = arguments[0];
  return arguments[0] = this, t.apply(null, arguments), this;
}
function Si() {
  return Array.from(this);
}
function Ci() {
  for (var t = this._groups, e = 0, n = t.length; e < n; ++e)
    for (var r = t[e], i = 0, o = r.length; i < o; ++i) {
      var s = r[i];
      if (s)
        return s;
    }
  return null;
}
function Ai() {
  let t = 0;
  for (const e of this)
    ++t;
  return t;
}
function ki() {
  return !this.node();
}
function Ni(t) {
  for (var e = this._groups, n = 0, r = e.length; n < r; ++n)
    for (var i = e[n], o = 0, s = i.length, u; o < s; ++o)
      (u = i[o]) && t.call(u, u.__data__, o, i);
  return this;
}
function $i(t) {
  return function() {
    this.removeAttribute(t);
  };
}
function Pi(t) {
  return function() {
    this.removeAttributeNS(t.space, t.local);
  };
}
function Oi(t, e) {
  return function() {
    this.setAttribute(t, e);
  };
}
function Fi(t, e) {
  return function() {
    this.setAttributeNS(t.space, t.local, e);
  };
}
function Mi(t, e) {
  return function() {
    var n = e.apply(this, arguments);
    n == null ? this.removeAttribute(t) : this.setAttribute(t, n);
  };
}
function Di(t, e) {
  return function() {
    var n = e.apply(this, arguments);
    n == null ? this.removeAttributeNS(t.space, t.local) : this.setAttributeNS(t.space, t.local, n);
  };
}
function Ii(t, e) {
  var n = Lt(t);
  if (arguments.length < 2) {
    var r = this.node();
    return n.local ? r.getAttributeNS(n.space, n.local) : r.getAttribute(n);
  }
  return this.each((e == null ? n.local ? Pi : $i : typeof e == "function" ? n.local ? Di : Mi : n.local ? Fi : Oi)(n, e));
}
function Sn(t) {
  return t.ownerDocument && t.ownerDocument.defaultView || t.document && t || t.defaultView;
}
function zi(t) {
  return function() {
    this.style.removeProperty(t);
  };
}
function Hi(t, e, n) {
  return function() {
    this.style.setProperty(t, e, n);
  };
}
function Vi(t, e, n) {
  return function() {
    var r = e.apply(this, arguments);
    r == null ? this.style.removeProperty(t) : this.style.setProperty(t, r, n);
  };
}
function Li(t, e, n) {
  return arguments.length > 1 ? this.each((e == null ? zi : typeof e == "function" ? Vi : Hi)(t, e, n == null ? "" : n)) : ot(this.node(), t);
}
function ot(t, e) {
  return t.style.getPropertyValue(e) || Sn(t).getComputedStyle(t, null).getPropertyValue(e);
}
function Yi(t) {
  return function() {
    delete this[t];
  };
}
function qi(t, e) {
  return function() {
    this[t] = e;
  };
}
function Bi(t, e) {
  return function() {
    var n = e.apply(this, arguments);
    n == null ? delete this[t] : this[t] = n;
  };
}
function Xi(t, e) {
  return arguments.length > 1 ? this.each((e == null ? Yi : typeof e == "function" ? Bi : qi)(t, e)) : this.node()[t];
}
function Cn(t) {
  return t.trim().split(/^|\s+/);
}
function _e(t) {
  return t.classList || new An(t);
}
function An(t) {
  this._node = t, this._names = Cn(t.getAttribute("class") || "");
}
An.prototype = {
  add: function(t) {
    var e = this._names.indexOf(t);
    e < 0 && (this._names.push(t), this._node.setAttribute("class", this._names.join(" ")));
  },
  remove: function(t) {
    var e = this._names.indexOf(t);
    e >= 0 && (this._names.splice(e, 1), this._node.setAttribute("class", this._names.join(" ")));
  },
  contains: function(t) {
    return this._names.indexOf(t) >= 0;
  }
};
function kn(t, e) {
  for (var n = _e(t), r = -1, i = e.length; ++r < i; )
    n.add(e[r]);
}
function Nn(t, e) {
  for (var n = _e(t), r = -1, i = e.length; ++r < i; )
    n.remove(e[r]);
}
function Ui(t) {
  return function() {
    kn(this, t);
  };
}
function Wi(t) {
  return function() {
    Nn(this, t);
  };
}
function ji(t, e) {
  return function() {
    (e.apply(this, arguments) ? kn : Nn)(this, t);
  };
}
function Gi(t, e) {
  var n = Cn(t + "");
  if (arguments.length < 2) {
    for (var r = _e(this.node()), i = -1, o = n.length; ++i < o; )
      if (!r.contains(n[i]))
        return !1;
    return !0;
  }
  return this.each((typeof e == "function" ? ji : e ? Ui : Wi)(n, e));
}
function Ji() {
  this.textContent = "";
}
function Ki(t) {
  return function() {
    this.textContent = t;
  };
}
function Zi(t) {
  return function() {
    var e = t.apply(this, arguments);
    this.textContent = e == null ? "" : e;
  };
}
function Qi(t) {
  return arguments.length ? this.each(t == null ? Ji : (typeof t == "function" ? Zi : Ki)(t)) : this.node().textContent;
}
function to() {
  this.innerHTML = "";
}
function eo(t) {
  return function() {
    this.innerHTML = t;
  };
}
function no(t) {
  return function() {
    var e = t.apply(this, arguments);
    this.innerHTML = e == null ? "" : e;
  };
}
function ro(t) {
  return arguments.length ? this.each(t == null ? to : (typeof t == "function" ? no : eo)(t)) : this.node().innerHTML;
}
function io() {
  this.nextSibling && this.parentNode.appendChild(this);
}
function oo() {
  return this.each(io);
}
function ao() {
  this.previousSibling && this.parentNode.insertBefore(this, this.parentNode.firstChild);
}
function so() {
  return this.each(ao);
}
function uo(t) {
  var e = typeof t == "function" ? t : xn(t);
  return this.select(function() {
    return this.appendChild(e.apply(this, arguments));
  });
}
function lo() {
  return null;
}
function co(t, e) {
  var n = typeof t == "function" ? t : xn(t), r = e == null ? lo : typeof e == "function" ? e : ge(e);
  return this.select(function() {
    return this.insertBefore(n.apply(this, arguments), r.apply(this, arguments) || null);
  });
}
function fo() {
  var t = this.parentNode;
  t && t.removeChild(this);
}
function ho() {
  return this.each(fo);
}
function po() {
  var t = this.cloneNode(!1), e = this.parentNode;
  return e ? e.insertBefore(t, this.nextSibling) : t;
}
function go() {
  var t = this.cloneNode(!0), e = this.parentNode;
  return e ? e.insertBefore(t, this.nextSibling) : t;
}
function _o(t) {
  return this.select(t ? go : po);
}
function yo(t) {
  return arguments.length ? this.property("__data__", t) : this.node().__data__;
}
function vo(t) {
  return function(e) {
    t.call(this, e, this.__data__);
  };
}
function wo(t) {
  return t.trim().split(/^|\s+/).map(function(e) {
    var n = "", r = e.indexOf(".");
    return r >= 0 && (n = e.slice(r + 1), e = e.slice(0, r)), { type: e, name: n };
  });
}
function mo(t) {
  return function() {
    var e = this.__on;
    if (!!e) {
      for (var n = 0, r = -1, i = e.length, o; n < i; ++n)
        o = e[n], (!t.type || o.type === t.type) && o.name === t.name ? this.removeEventListener(o.type, o.listener, o.options) : e[++r] = o;
      ++r ? e.length = r : delete this.__on;
    }
  };
}
function xo(t, e, n) {
  return function() {
    var r = this.__on, i, o = vo(e);
    if (r) {
      for (var s = 0, u = r.length; s < u; ++s)
        if ((i = r[s]).type === t.type && i.name === t.name) {
          this.removeEventListener(i.type, i.listener, i.options), this.addEventListener(i.type, i.listener = o, i.options = n), i.value = e;
          return;
        }
    }
    this.addEventListener(t.type, o, n), i = { type: t.type, name: t.name, value: e, listener: o, options: n }, r ? r.push(i) : this.__on = [i];
  };
}
function bo(t, e, n) {
  var r = wo(t + ""), i, o = r.length, s;
  if (arguments.length < 2) {
    var u = this.node().__on;
    if (u) {
      for (var l = 0, f = u.length, c; l < f; ++l)
        for (i = 0, c = u[l]; i < o; ++i)
          if ((s = r[i]).type === c.type && s.name === c.name)
            return c.value;
    }
    return;
  }
  for (u = e ? xo : mo, i = 0; i < o; ++i)
    this.each(u(r[i], e, n));
  return this;
}
function $n(t, e, n) {
  var r = Sn(t), i = r.CustomEvent;
  typeof i == "function" ? i = new i(e, n) : (i = r.document.createEvent("Event"), n ? (i.initEvent(e, n.bubbles, n.cancelable), i.detail = n.detail) : i.initEvent(e, !1, !1)), t.dispatchEvent(i);
}
function Eo(t, e) {
  return function() {
    return $n(this, t, e);
  };
}
function Ro(t, e) {
  return function() {
    return $n(this, t, e.apply(this, arguments));
  };
}
function To(t, e) {
  return this.each((typeof e == "function" ? Ro : Eo)(t, e));
}
function* So() {
  for (var t = this._groups, e = 0, n = t.length; e < n; ++e)
    for (var r = t[e], i = 0, o = r.length, s; i < o; ++i)
      (s = r[i]) && (yield s);
}
var Pn = [null];
function O(t, e) {
  this._groups = t, this._parents = e;
}
function wt() {
  return new O([[document.documentElement]], Pn);
}
function Co() {
  return this;
}
O.prototype = wt.prototype = {
  constructor: O,
  select: Zr,
  selectAll: ni,
  selectChild: ai,
  selectChildren: ci,
  filter: fi,
  data: yi,
  enter: hi,
  exit: wi,
  join: mi,
  merge: xi,
  selection: Co,
  order: bi,
  sort: Ei,
  call: Ti,
  nodes: Si,
  node: Ci,
  size: Ai,
  empty: ki,
  each: Ni,
  attr: Ii,
  style: Li,
  property: Xi,
  classed: Gi,
  text: Qi,
  html: ro,
  raise: oo,
  lower: so,
  append: uo,
  insert: co,
  remove: ho,
  clone: _o,
  datum: yo,
  on: bo,
  dispatch: To,
  [Symbol.iterator]: So
};
function Ke(t) {
  return typeof t == "string" ? new O([[document.querySelector(t)]], [document.documentElement]) : new O([[t]], Pn);
}
function ye(t, e, n) {
  t.prototype = e.prototype = n, n.constructor = t;
}
function On(t, e) {
  var n = Object.create(t.prototype);
  for (var r in e)
    n[r] = e[r];
  return n;
}
function mt() {
}
var gt = 0.7, Mt = 1 / gt, it = "\\s*([+-]?\\d+)\\s*", _t = "\\s*([+-]?(?:\\d*\\.)?\\d+(?:[eE][+-]?\\d+)?)\\s*", L = "\\s*([+-]?(?:\\d*\\.)?\\d+(?:[eE][+-]?\\d+)?)%\\s*", Ao = /^#([0-9a-f]{3,8})$/, ko = new RegExp(`^rgb\\(${it},${it},${it}\\)$`), No = new RegExp(`^rgb\\(${L},${L},${L}\\)$`), $o = new RegExp(`^rgba\\(${it},${it},${it},${_t}\\)$`), Po = new RegExp(`^rgba\\(${L},${L},${L},${_t}\\)$`), Oo = new RegExp(`^hsl\\(${_t},${L},${L}\\)$`), Fo = new RegExp(`^hsla\\(${_t},${L},${L},${_t}\\)$`), Ze = {
  aliceblue: 15792383,
  antiquewhite: 16444375,
  aqua: 65535,
  aquamarine: 8388564,
  azure: 15794175,
  beige: 16119260,
  bisque: 16770244,
  black: 0,
  blanchedalmond: 16772045,
  blue: 255,
  blueviolet: 9055202,
  brown: 10824234,
  burlywood: 14596231,
  cadetblue: 6266528,
  chartreuse: 8388352,
  chocolate: 13789470,
  coral: 16744272,
  cornflowerblue: 6591981,
  cornsilk: 16775388,
  crimson: 14423100,
  cyan: 65535,
  darkblue: 139,
  darkcyan: 35723,
  darkgoldenrod: 12092939,
  darkgray: 11119017,
  darkgreen: 25600,
  darkgrey: 11119017,
  darkkhaki: 12433259,
  darkmagenta: 9109643,
  darkolivegreen: 5597999,
  darkorange: 16747520,
  darkorchid: 10040012,
  darkred: 9109504,
  darksalmon: 15308410,
  darkseagreen: 9419919,
  darkslateblue: 4734347,
  darkslategray: 3100495,
  darkslategrey: 3100495,
  darkturquoise: 52945,
  darkviolet: 9699539,
  deeppink: 16716947,
  deepskyblue: 49151,
  dimgray: 6908265,
  dimgrey: 6908265,
  dodgerblue: 2003199,
  firebrick: 11674146,
  floralwhite: 16775920,
  forestgreen: 2263842,
  fuchsia: 16711935,
  gainsboro: 14474460,
  ghostwhite: 16316671,
  gold: 16766720,
  goldenrod: 14329120,
  gray: 8421504,
  green: 32768,
  greenyellow: 11403055,
  grey: 8421504,
  honeydew: 15794160,
  hotpink: 16738740,
  indianred: 13458524,
  indigo: 4915330,
  ivory: 16777200,
  khaki: 15787660,
  lavender: 15132410,
  lavenderblush: 16773365,
  lawngreen: 8190976,
  lemonchiffon: 16775885,
  lightblue: 11393254,
  lightcoral: 15761536,
  lightcyan: 14745599,
  lightgoldenrodyellow: 16448210,
  lightgray: 13882323,
  lightgreen: 9498256,
  lightgrey: 13882323,
  lightpink: 16758465,
  lightsalmon: 16752762,
  lightseagreen: 2142890,
  lightskyblue: 8900346,
  lightslategray: 7833753,
  lightslategrey: 7833753,
  lightsteelblue: 11584734,
  lightyellow: 16777184,
  lime: 65280,
  limegreen: 3329330,
  linen: 16445670,
  magenta: 16711935,
  maroon: 8388608,
  mediumaquamarine: 6737322,
  mediumblue: 205,
  mediumorchid: 12211667,
  mediumpurple: 9662683,
  mediumseagreen: 3978097,
  mediumslateblue: 8087790,
  mediumspringgreen: 64154,
  mediumturquoise: 4772300,
  mediumvioletred: 13047173,
  midnightblue: 1644912,
  mintcream: 16121850,
  mistyrose: 16770273,
  moccasin: 16770229,
  navajowhite: 16768685,
  navy: 128,
  oldlace: 16643558,
  olive: 8421376,
  olivedrab: 7048739,
  orange: 16753920,
  orangered: 16729344,
  orchid: 14315734,
  palegoldenrod: 15657130,
  palegreen: 10025880,
  paleturquoise: 11529966,
  palevioletred: 14381203,
  papayawhip: 16773077,
  peachpuff: 16767673,
  peru: 13468991,
  pink: 16761035,
  plum: 14524637,
  powderblue: 11591910,
  purple: 8388736,
  rebeccapurple: 6697881,
  red: 16711680,
  rosybrown: 12357519,
  royalblue: 4286945,
  saddlebrown: 9127187,
  salmon: 16416882,
  sandybrown: 16032864,
  seagreen: 3050327,
  seashell: 16774638,
  sienna: 10506797,
  silver: 12632256,
  skyblue: 8900331,
  slateblue: 6970061,
  slategray: 7372944,
  slategrey: 7372944,
  snow: 16775930,
  springgreen: 65407,
  steelblue: 4620980,
  tan: 13808780,
  teal: 32896,
  thistle: 14204888,
  tomato: 16737095,
  turquoise: 4251856,
  violet: 15631086,
  wheat: 16113331,
  white: 16777215,
  whitesmoke: 16119285,
  yellow: 16776960,
  yellowgreen: 10145074
};
ye(mt, yt, {
  copy(t) {
    return Object.assign(new this.constructor(), this, t);
  },
  displayable() {
    return this.rgb().displayable();
  },
  hex: Qe,
  formatHex: Qe,
  formatHex8: Mo,
  formatHsl: Do,
  formatRgb: tn,
  toString: tn
});
function Qe() {
  return this.rgb().formatHex();
}
function Mo() {
  return this.rgb().formatHex8();
}
function Do() {
  return Fn(this).formatHsl();
}
function tn() {
  return this.rgb().formatRgb();
}
function yt(t) {
  var e, n;
  return t = (t + "").trim().toLowerCase(), (e = Ao.exec(t)) ? (n = e[1].length, e = parseInt(e[1], 16), n === 6 ? en(e) : n === 3 ? new P(e >> 8 & 15 | e >> 4 & 240, e >> 4 & 15 | e & 240, (e & 15) << 4 | e & 15, 1) : n === 8 ? At(e >> 24 & 255, e >> 16 & 255, e >> 8 & 255, (e & 255) / 255) : n === 4 ? At(e >> 12 & 15 | e >> 8 & 240, e >> 8 & 15 | e >> 4 & 240, e >> 4 & 15 | e & 240, ((e & 15) << 4 | e & 15) / 255) : null) : (e = ko.exec(t)) ? new P(e[1], e[2], e[3], 1) : (e = No.exec(t)) ? new P(e[1] * 255 / 100, e[2] * 255 / 100, e[3] * 255 / 100, 1) : (e = $o.exec(t)) ? At(e[1], e[2], e[3], e[4]) : (e = Po.exec(t)) ? At(e[1] * 255 / 100, e[2] * 255 / 100, e[3] * 255 / 100, e[4]) : (e = Oo.exec(t)) ? on(e[1], e[2] / 100, e[3] / 100, 1) : (e = Fo.exec(t)) ? on(e[1], e[2] / 100, e[3] / 100, e[4]) : Ze.hasOwnProperty(t) ? en(Ze[t]) : t === "transparent" ? new P(NaN, NaN, NaN, 0) : null;
}
function en(t) {
  return new P(t >> 16 & 255, t >> 8 & 255, t & 255, 1);
}
function At(t, e, n, r) {
  return r <= 0 && (t = e = n = NaN), new P(t, e, n, r);
}
function Io(t) {
  return t instanceof mt || (t = yt(t)), t ? (t = t.rgb(), new P(t.r, t.g, t.b, t.opacity)) : new P();
}
function re(t, e, n, r) {
  return arguments.length === 1 ? Io(t) : new P(t, e, n, r == null ? 1 : r);
}
function P(t, e, n, r) {
  this.r = +t, this.g = +e, this.b = +n, this.opacity = +r;
}
ye(P, re, On(mt, {
  brighter(t) {
    return t = t == null ? Mt : Math.pow(Mt, t), new P(this.r * t, this.g * t, this.b * t, this.opacity);
  },
  darker(t) {
    return t = t == null ? gt : Math.pow(gt, t), new P(this.r * t, this.g * t, this.b * t, this.opacity);
  },
  rgb() {
    return this;
  },
  clamp() {
    return new P(Z(this.r), Z(this.g), Z(this.b), Dt(this.opacity));
  },
  displayable() {
    return -0.5 <= this.r && this.r < 255.5 && -0.5 <= this.g && this.g < 255.5 && -0.5 <= this.b && this.b < 255.5 && 0 <= this.opacity && this.opacity <= 1;
  },
  hex: nn,
  formatHex: nn,
  formatHex8: zo,
  formatRgb: rn,
  toString: rn
}));
function nn() {
  return `#${K(this.r)}${K(this.g)}${K(this.b)}`;
}
function zo() {
  return `#${K(this.r)}${K(this.g)}${K(this.b)}${K((isNaN(this.opacity) ? 1 : this.opacity) * 255)}`;
}
function rn() {
  const t = Dt(this.opacity);
  return `${t === 1 ? "rgb(" : "rgba("}${Z(this.r)}, ${Z(this.g)}, ${Z(this.b)}${t === 1 ? ")" : `, ${t})`}`;
}
function Dt(t) {
  return isNaN(t) ? 1 : Math.max(0, Math.min(1, t));
}
function Z(t) {
  return Math.max(0, Math.min(255, Math.round(t) || 0));
}
function K(t) {
  return t = Z(t), (t < 16 ? "0" : "") + t.toString(16);
}
function on(t, e, n, r) {
  return r <= 0 ? t = e = n = NaN : n <= 0 || n >= 1 ? t = e = NaN : e <= 0 && (t = NaN), new I(t, e, n, r);
}
function Fn(t) {
  if (t instanceof I)
    return new I(t.h, t.s, t.l, t.opacity);
  if (t instanceof mt || (t = yt(t)), !t)
    return new I();
  if (t instanceof I)
    return t;
  t = t.rgb();
  var e = t.r / 255, n = t.g / 255, r = t.b / 255, i = Math.min(e, n, r), o = Math.max(e, n, r), s = NaN, u = o - i, l = (o + i) / 2;
  return u ? (e === o ? s = (n - r) / u + (n < r) * 6 : n === o ? s = (r - e) / u + 2 : s = (e - n) / u + 4, u /= l < 0.5 ? o + i : 2 - o - i, s *= 60) : u = l > 0 && l < 1 ? 0 : s, new I(s, u, l, t.opacity);
}
function Ho(t, e, n, r) {
  return arguments.length === 1 ? Fn(t) : new I(t, e, n, r == null ? 1 : r);
}
function I(t, e, n, r) {
  this.h = +t, this.s = +e, this.l = +n, this.opacity = +r;
}
ye(I, Ho, On(mt, {
  brighter(t) {
    return t = t == null ? Mt : Math.pow(Mt, t), new I(this.h, this.s, this.l * t, this.opacity);
  },
  darker(t) {
    return t = t == null ? gt : Math.pow(gt, t), new I(this.h, this.s, this.l * t, this.opacity);
  },
  rgb() {
    var t = this.h % 360 + (this.h < 0) * 360, e = isNaN(t) || isNaN(this.s) ? 0 : this.s, n = this.l, r = n + (n < 0.5 ? n : 1 - n) * e, i = 2 * n - r;
    return new P(
      Qt(t >= 240 ? t - 240 : t + 120, i, r),
      Qt(t, i, r),
      Qt(t < 120 ? t + 240 : t - 120, i, r),
      this.opacity
    );
  },
  clamp() {
    return new I(an(this.h), kt(this.s), kt(this.l), Dt(this.opacity));
  },
  displayable() {
    return (0 <= this.s && this.s <= 1 || isNaN(this.s)) && 0 <= this.l && this.l <= 1 && 0 <= this.opacity && this.opacity <= 1;
  },
  formatHsl() {
    const t = Dt(this.opacity);
    return `${t === 1 ? "hsl(" : "hsla("}${an(this.h)}, ${kt(this.s) * 100}%, ${kt(this.l) * 100}%${t === 1 ? ")" : `, ${t})`}`;
  }
}));
function an(t) {
  return t = (t || 0) % 360, t < 0 ? t + 360 : t;
}
function kt(t) {
  return Math.max(0, Math.min(1, t || 0));
}
function Qt(t, e, n) {
  return (t < 60 ? e + (n - e) * t / 60 : t < 180 ? n : t < 240 ? e + (n - e) * (240 - t) / 60 : e) * 255;
}
const Mn = (t) => () => t;
function Vo(t, e) {
  return function(n) {
    return t + n * e;
  };
}
function Lo(t, e, n) {
  return t = Math.pow(t, n), e = Math.pow(e, n) - t, n = 1 / n, function(r) {
    return Math.pow(t + r * e, n);
  };
}
function Yo(t) {
  return (t = +t) == 1 ? Dn : function(e, n) {
    return n - e ? Lo(e, n, t) : Mn(isNaN(e) ? n : e);
  };
}
function Dn(t, e) {
  var n = e - t;
  return n ? Vo(t, n) : Mn(isNaN(t) ? e : t);
}
const sn = function t(e) {
  var n = Yo(e);
  function r(i, o) {
    var s = n((i = re(i)).r, (o = re(o)).r), u = n(i.g, o.g), l = n(i.b, o.b), f = Dn(i.opacity, o.opacity);
    return function(c) {
      return i.r = s(c), i.g = u(c), i.b = l(c), i.opacity = f(c), i + "";
    };
  }
  return r.gamma = t, r;
}(1);
function W(t, e) {
  return t = +t, e = +e, function(n) {
    return t * (1 - n) + e * n;
  };
}
var ie = /[-+]?(?:\d+\.?\d*|\.?\d+)(?:[eE][-+]?\d+)?/g, te = new RegExp(ie.source, "g");
function qo(t) {
  return function() {
    return t;
  };
}
function Bo(t) {
  return function(e) {
    return t(e) + "";
  };
}
function Xo(t, e) {
  var n = ie.lastIndex = te.lastIndex = 0, r, i, o, s = -1, u = [], l = [];
  for (t = t + "", e = e + ""; (r = ie.exec(t)) && (i = te.exec(e)); )
    (o = i.index) > n && (o = e.slice(n, o), u[s] ? u[s] += o : u[++s] = o), (r = r[0]) === (i = i[0]) ? u[s] ? u[s] += i : u[++s] = i : (u[++s] = null, l.push({ i: s, x: W(r, i) })), n = te.lastIndex;
  return n < e.length && (o = e.slice(n), u[s] ? u[s] += o : u[++s] = o), u.length < 2 ? l[0] ? Bo(l[0].x) : qo(e) : (e = l.length, function(f) {
    for (var c = 0, p; c < e; ++c)
      u[(p = l[c]).i] = p.x(f);
    return u.join("");
  });
}
var un = 180 / Math.PI, oe = {
  translateX: 0,
  translateY: 0,
  rotate: 0,
  skewX: 0,
  scaleX: 1,
  scaleY: 1
};
function In(t, e, n, r, i, o) {
  var s, u, l;
  return (s = Math.sqrt(t * t + e * e)) && (t /= s, e /= s), (l = t * n + e * r) && (n -= t * l, r -= e * l), (u = Math.sqrt(n * n + r * r)) && (n /= u, r /= u, l /= u), t * r < e * n && (t = -t, e = -e, l = -l, s = -s), {
    translateX: i,
    translateY: o,
    rotate: Math.atan2(e, t) * un,
    skewX: Math.atan(l) * un,
    scaleX: s,
    scaleY: u
  };
}
var Nt;
function Uo(t) {
  const e = new (typeof DOMMatrix == "function" ? DOMMatrix : WebKitCSSMatrix)(t + "");
  return e.isIdentity ? oe : In(e.a, e.b, e.c, e.d, e.e, e.f);
}
function Wo(t) {
  return t == null || (Nt || (Nt = document.createElementNS("http://www.w3.org/2000/svg", "g")), Nt.setAttribute("transform", t), !(t = Nt.transform.baseVal.consolidate())) ? oe : (t = t.matrix, In(t.a, t.b, t.c, t.d, t.e, t.f));
}
function zn(t, e, n, r) {
  function i(f) {
    return f.length ? f.pop() + " " : "";
  }
  function o(f, c, p, g, _, E) {
    if (f !== p || c !== g) {
      var C = _.push("translate(", null, e, null, n);
      E.push({ i: C - 4, x: W(f, p) }, { i: C - 2, x: W(c, g) });
    } else
      (p || g) && _.push("translate(" + p + e + g + n);
  }
  function s(f, c, p, g) {
    f !== c ? (f - c > 180 ? c += 360 : c - f > 180 && (f += 360), g.push({ i: p.push(i(p) + "rotate(", null, r) - 2, x: W(f, c) })) : c && p.push(i(p) + "rotate(" + c + r);
  }
  function u(f, c, p, g) {
    f !== c ? g.push({ i: p.push(i(p) + "skewX(", null, r) - 2, x: W(f, c) }) : c && p.push(i(p) + "skewX(" + c + r);
  }
  function l(f, c, p, g, _, E) {
    if (f !== p || c !== g) {
      var C = _.push(i(_) + "scale(", null, ",", null, ")");
      E.push({ i: C - 4, x: W(f, p) }, { i: C - 2, x: W(c, g) });
    } else
      (p !== 1 || g !== 1) && _.push(i(_) + "scale(" + p + "," + g + ")");
  }
  return function(f, c) {
    var p = [], g = [];
    return f = t(f), c = t(c), o(f.translateX, f.translateY, c.translateX, c.translateY, p, g), s(f.rotate, c.rotate, p, g), u(f.skewX, c.skewX, p, g), l(f.scaleX, f.scaleY, c.scaleX, c.scaleY, p, g), f = c = null, function(_) {
      for (var E = -1, C = g.length, F; ++E < C; )
        p[(F = g[E]).i] = F.x(_);
      return p.join("");
    };
  };
}
var jo = zn(Uo, "px, ", "px)", "deg)"), Go = zn(Wo, ", ", ")", ")"), at = 0, pt = 0, ht = 0, Hn = 1e3, It, dt, zt = 0, Q = 0, Yt = 0, vt = typeof performance == "object" && performance.now ? performance : Date, Vn = typeof window == "object" && window.requestAnimationFrame ? window.requestAnimationFrame.bind(window) : function(t) {
  setTimeout(t, 17);
};
function ve() {
  return Q || (Vn(Jo), Q = vt.now() + Yt);
}
function Jo() {
  Q = 0;
}
function Ht() {
  this._call = this._time = this._next = null;
}
Ht.prototype = Ln.prototype = {
  constructor: Ht,
  restart: function(t, e, n) {
    if (typeof t != "function")
      throw new TypeError("callback is not a function");
    n = (n == null ? ve() : +n) + (e == null ? 0 : +e), !this._next && dt !== this && (dt ? dt._next = this : It = this, dt = this), this._call = t, this._time = n, ae();
  },
  stop: function() {
    this._call && (this._call = null, this._time = 1 / 0, ae());
  }
};
function Ln(t, e, n) {
  var r = new Ht();
  return r.restart(t, e, n), r;
}
function Ko() {
  ve(), ++at;
  for (var t = It, e; t; )
    (e = Q - t._time) >= 0 && t._call.call(void 0, e), t = t._next;
  --at;
}
function ln() {
  Q = (zt = vt.now()) + Yt, at = pt = 0;
  try {
    Ko();
  } finally {
    at = 0, Qo(), Q = 0;
  }
}
function Zo() {
  var t = vt.now(), e = t - zt;
  e > Hn && (Yt -= e, zt = t);
}
function Qo() {
  for (var t, e = It, n, r = 1 / 0; e; )
    e._call ? (r > e._time && (r = e._time), t = e, e = e._next) : (n = e._next, e._next = null, e = t ? t._next = n : It = n);
  dt = t, ae(r);
}
function ae(t) {
  if (!at) {
    pt && (pt = clearTimeout(pt));
    var e = t - Q;
    e > 24 ? (t < 1 / 0 && (pt = setTimeout(ln, t - vt.now() - Yt)), ht && (ht = clearInterval(ht))) : (ht || (zt = vt.now(), ht = setInterval(Zo, Hn)), at = 1, Vn(ln));
  }
}
function cn(t, e, n) {
  var r = new Ht();
  return e = e == null ? 0 : +e, r.restart((i) => {
    r.stop(), t(i + e);
  }, e, n), r;
}
var ta = mn("start", "end", "cancel", "interrupt"), ea = [], Yn = 0, fn = 1, se = 2, Pt = 3, hn = 4, ue = 5, Ot = 6;
function qt(t, e, n, r, i, o) {
  var s = t.__transition;
  if (!s)
    t.__transition = {};
  else if (n in s)
    return;
  na(t, n, {
    name: e,
    index: r,
    group: i,
    on: ta,
    tween: ea,
    time: o.time,
    delay: o.delay,
    duration: o.duration,
    ease: o.ease,
    timer: null,
    state: Yn
  });
}
function we(t, e) {
  var n = z(t, e);
  if (n.state > Yn)
    throw new Error("too late; already scheduled");
  return n;
}
function Y(t, e) {
  var n = z(t, e);
  if (n.state > Pt)
    throw new Error("too late; already running");
  return n;
}
function z(t, e) {
  var n = t.__transition;
  if (!n || !(n = n[e]))
    throw new Error("transition not found");
  return n;
}
function na(t, e, n) {
  var r = t.__transition, i;
  r[e] = n, n.timer = Ln(o, 0, n.time);
  function o(f) {
    n.state = fn, n.timer.restart(s, n.delay, n.time), n.delay <= f && s(f - n.delay);
  }
  function s(f) {
    var c, p, g, _;
    if (n.state !== fn)
      return l();
    for (c in r)
      if (_ = r[c], _.name === n.name) {
        if (_.state === Pt)
          return cn(s);
        _.state === hn ? (_.state = Ot, _.timer.stop(), _.on.call("interrupt", t, t.__data__, _.index, _.group), delete r[c]) : +c < e && (_.state = Ot, _.timer.stop(), _.on.call("cancel", t, t.__data__, _.index, _.group), delete r[c]);
      }
    if (cn(function() {
      n.state === Pt && (n.state = hn, n.timer.restart(u, n.delay, n.time), u(f));
    }), n.state = se, n.on.call("start", t, t.__data__, n.index, n.group), n.state === se) {
      for (n.state = Pt, i = new Array(g = n.tween.length), c = 0, p = -1; c < g; ++c)
        (_ = n.tween[c].value.call(t, t.__data__, n.index, n.group)) && (i[++p] = _);
      i.length = p + 1;
    }
  }
  function u(f) {
    for (var c = f < n.duration ? n.ease.call(null, f / n.duration) : (n.timer.restart(l), n.state = ue, 1), p = -1, g = i.length; ++p < g; )
      i[p].call(t, c);
    n.state === ue && (n.on.call("end", t, t.__data__, n.index, n.group), l());
  }
  function l() {
    n.state = Ot, n.timer.stop(), delete r[e];
    for (var f in r)
      return;
    delete t.__transition;
  }
}
function ra(t, e) {
  var n = t.__transition, r, i, o = !0, s;
  if (!!n) {
    e = e == null ? null : e + "";
    for (s in n) {
      if ((r = n[s]).name !== e) {
        o = !1;
        continue;
      }
      i = r.state > se && r.state < ue, r.state = Ot, r.timer.stop(), r.on.call(i ? "interrupt" : "cancel", t, t.__data__, r.index, r.group), delete n[s];
    }
    o && delete t.__transition;
  }
}
function ia(t) {
  return this.each(function() {
    ra(this, t);
  });
}
function oa(t, e) {
  var n, r;
  return function() {
    var i = Y(this, t), o = i.tween;
    if (o !== n) {
      r = n = o;
      for (var s = 0, u = r.length; s < u; ++s)
        if (r[s].name === e) {
          r = r.slice(), r.splice(s, 1);
          break;
        }
    }
    i.tween = r;
  };
}
function aa(t, e, n) {
  var r, i;
  if (typeof n != "function")
    throw new Error();
  return function() {
    var o = Y(this, t), s = o.tween;
    if (s !== r) {
      i = (r = s).slice();
      for (var u = { name: e, value: n }, l = 0, f = i.length; l < f; ++l)
        if (i[l].name === e) {
          i[l] = u;
          break;
        }
      l === f && i.push(u);
    }
    o.tween = i;
  };
}
function sa(t, e) {
  var n = this._id;
  if (t += "", arguments.length < 2) {
    for (var r = z(this.node(), n).tween, i = 0, o = r.length, s; i < o; ++i)
      if ((s = r[i]).name === t)
        return s.value;
    return null;
  }
  return this.each((e == null ? oa : aa)(n, t, e));
}
function me(t, e, n) {
  var r = t._id;
  return t.each(function() {
    var i = Y(this, r);
    (i.value || (i.value = {}))[e] = n.apply(this, arguments);
  }), function(i) {
    return z(i, r).value[e];
  };
}
function qn(t, e) {
  var n;
  return (typeof e == "number" ? W : e instanceof yt ? sn : (n = yt(e)) ? (e = n, sn) : Xo)(t, e);
}
function ua(t) {
  return function() {
    this.removeAttribute(t);
  };
}
function la(t) {
  return function() {
    this.removeAttributeNS(t.space, t.local);
  };
}
function ca(t, e, n) {
  var r, i = n + "", o;
  return function() {
    var s = this.getAttribute(t);
    return s === i ? null : s === r ? o : o = e(r = s, n);
  };
}
function fa(t, e, n) {
  var r, i = n + "", o;
  return function() {
    var s = this.getAttributeNS(t.space, t.local);
    return s === i ? null : s === r ? o : o = e(r = s, n);
  };
}
function ha(t, e, n) {
  var r, i, o;
  return function() {
    var s, u = n(this), l;
    return u == null ? void this.removeAttribute(t) : (s = this.getAttribute(t), l = u + "", s === l ? null : s === r && l === i ? o : (i = l, o = e(r = s, u)));
  };
}
function pa(t, e, n) {
  var r, i, o;
  return function() {
    var s, u = n(this), l;
    return u == null ? void this.removeAttributeNS(t.space, t.local) : (s = this.getAttributeNS(t.space, t.local), l = u + "", s === l ? null : s === r && l === i ? o : (i = l, o = e(r = s, u)));
  };
}
function da(t, e) {
  var n = Lt(t), r = n === "transform" ? Go : qn;
  return this.attrTween(t, typeof e == "function" ? (n.local ? pa : ha)(n, r, me(this, "attr." + t, e)) : e == null ? (n.local ? la : ua)(n) : (n.local ? fa : ca)(n, r, e));
}
function ga(t, e) {
  return function(n) {
    this.setAttribute(t, e.call(this, n));
  };
}
function _a(t, e) {
  return function(n) {
    this.setAttributeNS(t.space, t.local, e.call(this, n));
  };
}
function ya(t, e) {
  var n, r;
  function i() {
    var o = e.apply(this, arguments);
    return o !== r && (n = (r = o) && _a(t, o)), n;
  }
  return i._value = e, i;
}
function va(t, e) {
  var n, r;
  function i() {
    var o = e.apply(this, arguments);
    return o !== r && (n = (r = o) && ga(t, o)), n;
  }
  return i._value = e, i;
}
function wa(t, e) {
  var n = "attr." + t;
  if (arguments.length < 2)
    return (n = this.tween(n)) && n._value;
  if (e == null)
    return this.tween(n, null);
  if (typeof e != "function")
    throw new Error();
  var r = Lt(t);
  return this.tween(n, (r.local ? ya : va)(r, e));
}
function ma(t, e) {
  return function() {
    we(this, t).delay = +e.apply(this, arguments);
  };
}
function xa(t, e) {
  return e = +e, function() {
    we(this, t).delay = e;
  };
}
function ba(t) {
  var e = this._id;
  return arguments.length ? this.each((typeof t == "function" ? ma : xa)(e, t)) : z(this.node(), e).delay;
}
function Ea(t, e) {
  return function() {
    Y(this, t).duration = +e.apply(this, arguments);
  };
}
function Ra(t, e) {
  return e = +e, function() {
    Y(this, t).duration = e;
  };
}
function Ta(t) {
  var e = this._id;
  return arguments.length ? this.each((typeof t == "function" ? Ea : Ra)(e, t)) : z(this.node(), e).duration;
}
function Sa(t, e) {
  if (typeof e != "function")
    throw new Error();
  return function() {
    Y(this, t).ease = e;
  };
}
function Ca(t) {
  var e = this._id;
  return arguments.length ? this.each(Sa(e, t)) : z(this.node(), e).ease;
}
function Aa(t, e) {
  return function() {
    var n = e.apply(this, arguments);
    if (typeof n != "function")
      throw new Error();
    Y(this, t).ease = n;
  };
}
function ka(t) {
  if (typeof t != "function")
    throw new Error();
  return this.each(Aa(this._id, t));
}
function Na(t) {
  typeof t != "function" && (t = En(t));
  for (var e = this._groups, n = e.length, r = new Array(n), i = 0; i < n; ++i)
    for (var o = e[i], s = o.length, u = r[i] = [], l, f = 0; f < s; ++f)
      (l = o[f]) && t.call(l, l.__data__, f, o) && u.push(l);
  return new U(r, this._parents, this._name, this._id);
}
function $a(t) {
  if (t._id !== this._id)
    throw new Error();
  for (var e = this._groups, n = t._groups, r = e.length, i = n.length, o = Math.min(r, i), s = new Array(r), u = 0; u < o; ++u)
    for (var l = e[u], f = n[u], c = l.length, p = s[u] = new Array(c), g, _ = 0; _ < c; ++_)
      (g = l[_] || f[_]) && (p[_] = g);
  for (; u < r; ++u)
    s[u] = e[u];
  return new U(s, this._parents, this._name, this._id);
}
function Pa(t) {
  return (t + "").trim().split(/^|\s+/).every(function(e) {
    var n = e.indexOf(".");
    return n >= 0 && (e = e.slice(0, n)), !e || e === "start";
  });
}
function Oa(t, e, n) {
  var r, i, o = Pa(e) ? we : Y;
  return function() {
    var s = o(this, t), u = s.on;
    u !== r && (i = (r = u).copy()).on(e, n), s.on = i;
  };
}
function Fa(t, e) {
  var n = this._id;
  return arguments.length < 2 ? z(this.node(), n).on.on(t) : this.each(Oa(n, t, e));
}
function Ma(t) {
  return function() {
    var e = this.parentNode;
    for (var n in this.__transition)
      if (+n !== t)
        return;
    e && e.removeChild(this);
  };
}
function Da() {
  return this.on("end.remove", Ma(this._id));
}
function Ia(t) {
  var e = this._name, n = this._id;
  typeof t != "function" && (t = ge(t));
  for (var r = this._groups, i = r.length, o = new Array(i), s = 0; s < i; ++s)
    for (var u = r[s], l = u.length, f = o[s] = new Array(l), c, p, g = 0; g < l; ++g)
      (c = u[g]) && (p = t.call(c, c.__data__, g, u)) && ("__data__" in c && (p.__data__ = c.__data__), f[g] = p, qt(f[g], e, n, g, f, z(c, n)));
  return new U(o, this._parents, e, n);
}
function za(t) {
  var e = this._name, n = this._id;
  typeof t != "function" && (t = bn(t));
  for (var r = this._groups, i = r.length, o = [], s = [], u = 0; u < i; ++u)
    for (var l = r[u], f = l.length, c, p = 0; p < f; ++p)
      if (c = l[p]) {
        for (var g = t.call(c, c.__data__, p, l), _, E = z(c, n), C = 0, F = g.length; C < F; ++C)
          (_ = g[C]) && qt(_, e, n, C, g, E);
        o.push(g), s.push(c);
      }
  return new U(o, s, e, n);
}
var Ha = wt.prototype.constructor;
function Va() {
  return new Ha(this._groups, this._parents);
}
function La(t, e) {
  var n, r, i;
  return function() {
    var o = ot(this, t), s = (this.style.removeProperty(t), ot(this, t));
    return o === s ? null : o === n && s === r ? i : i = e(n = o, r = s);
  };
}
function Bn(t) {
  return function() {
    this.style.removeProperty(t);
  };
}
function Ya(t, e, n) {
  var r, i = n + "", o;
  return function() {
    var s = ot(this, t);
    return s === i ? null : s === r ? o : o = e(r = s, n);
  };
}
function qa(t, e, n) {
  var r, i, o;
  return function() {
    var s = ot(this, t), u = n(this), l = u + "";
    return u == null && (l = u = (this.style.removeProperty(t), ot(this, t))), s === l ? null : s === r && l === i ? o : (i = l, o = e(r = s, u));
  };
}
function Ba(t, e) {
  var n, r, i, o = "style." + e, s = "end." + o, u;
  return function() {
    var l = Y(this, t), f = l.on, c = l.value[o] == null ? u || (u = Bn(e)) : void 0;
    (f !== n || i !== c) && (r = (n = f).copy()).on(s, i = c), l.on = r;
  };
}
function Xa(t, e, n) {
  var r = (t += "") == "transform" ? jo : qn;
  return e == null ? this.styleTween(t, La(t, r)).on("end.style." + t, Bn(t)) : typeof e == "function" ? this.styleTween(t, qa(t, r, me(this, "style." + t, e))).each(Ba(this._id, t)) : this.styleTween(t, Ya(t, r, e), n).on("end.style." + t, null);
}
function Ua(t, e, n) {
  return function(r) {
    this.style.setProperty(t, e.call(this, r), n);
  };
}
function Wa(t, e, n) {
  var r, i;
  function o() {
    var s = e.apply(this, arguments);
    return s !== i && (r = (i = s) && Ua(t, s, n)), r;
  }
  return o._value = e, o;
}
function ja(t, e, n) {
  var r = "style." + (t += "");
  if (arguments.length < 2)
    return (r = this.tween(r)) && r._value;
  if (e == null)
    return this.tween(r, null);
  if (typeof e != "function")
    throw new Error();
  return this.tween(r, Wa(t, e, n == null ? "" : n));
}
function Ga(t) {
  return function() {
    this.textContent = t;
  };
}
function Ja(t) {
  return function() {
    var e = t(this);
    this.textContent = e == null ? "" : e;
  };
}
function Ka(t) {
  return this.tween("text", typeof t == "function" ? Ja(me(this, "text", t)) : Ga(t == null ? "" : t + ""));
}
function Za(t) {
  return function(e) {
    this.textContent = t.call(this, e);
  };
}
function Qa(t) {
  var e, n;
  function r() {
    var i = t.apply(this, arguments);
    return i !== n && (e = (n = i) && Za(i)), e;
  }
  return r._value = t, r;
}
function ts(t) {
  var e = "text";
  if (arguments.length < 1)
    return (e = this.tween(e)) && e._value;
  if (t == null)
    return this.tween(e, null);
  if (typeof t != "function")
    throw new Error();
  return this.tween(e, Qa(t));
}
function es() {
  for (var t = this._name, e = this._id, n = Xn(), r = this._groups, i = r.length, o = 0; o < i; ++o)
    for (var s = r[o], u = s.length, l, f = 0; f < u; ++f)
      if (l = s[f]) {
        var c = z(l, e);
        qt(l, t, n, f, s, {
          time: c.time + c.delay + c.duration,
          delay: 0,
          duration: c.duration,
          ease: c.ease
        });
      }
  return new U(r, this._parents, t, n);
}
function ns() {
  var t, e, n = this, r = n._id, i = n.size();
  return new Promise(function(o, s) {
    var u = { value: s }, l = { value: function() {
      --i === 0 && o();
    } };
    n.each(function() {
      var f = Y(this, r), c = f.on;
      c !== t && (e = (t = c).copy(), e._.cancel.push(u), e._.interrupt.push(u), e._.end.push(l)), f.on = e;
    }), i === 0 && o();
  });
}
var rs = 0;
function U(t, e, n, r) {
  this._groups = t, this._parents = e, this._name = n, this._id = r;
}
function Xn() {
  return ++rs;
}
var X = wt.prototype;
U.prototype = {
  constructor: U,
  select: Ia,
  selectAll: za,
  selectChild: X.selectChild,
  selectChildren: X.selectChildren,
  filter: Na,
  merge: $a,
  selection: Va,
  transition: es,
  call: X.call,
  nodes: X.nodes,
  node: X.node,
  size: X.size,
  empty: X.empty,
  each: X.each,
  on: Fa,
  attr: da,
  attrTween: wa,
  style: Xa,
  styleTween: ja,
  text: Ka,
  textTween: ts,
  remove: Da,
  tween: sa,
  delay: ba,
  duration: Ta,
  ease: Ca,
  easeVarying: ka,
  end: ns,
  [Symbol.iterator]: X[Symbol.iterator]
};
const is = (t) => +t;
function os(t) {
  return ((t *= 2) <= 1 ? t * t * t : (t -= 2) * t * t + 2) / 2;
}
var as = {
  time: null,
  delay: 0,
  duration: 250,
  ease: os
};
function ss(t, e) {
  for (var n; !(n = t.__transition) || !(n = n[e]); )
    if (!(t = t.parentNode))
      throw new Error(`transition ${e} not found`);
  return n;
}
function us(t) {
  var e, n;
  t instanceof U ? (e = t._id, t = t._name) : (e = Xn(), (n = as).time = ve(), t = t == null ? null : t + "");
  for (var r = this._groups, i = r.length, o = 0; o < i; ++o)
    for (var s = r[o], u = s.length, l, f = 0; f < u; ++f)
      (l = s[f]) && qt(l, t, e, f, s, n || ss(l, e));
  return new U(r, this._parents, t, e);
}
wt.prototype.interrupt = ia;
wt.prototype.transition = us;
const le = Math.PI, ce = 2 * le, J = 1e-6, ls = ce - J;
function fe() {
  this._x0 = this._y0 = this._x1 = this._y1 = null, this._ = "";
}
function Un() {
  return new fe();
}
fe.prototype = Un.prototype = {
  constructor: fe,
  moveTo: function(t, e) {
    this._ += "M" + (this._x0 = this._x1 = +t) + "," + (this._y0 = this._y1 = +e);
  },
  closePath: function() {
    this._x1 !== null && (this._x1 = this._x0, this._y1 = this._y0, this._ += "Z");
  },
  lineTo: function(t, e) {
    this._ += "L" + (this._x1 = +t) + "," + (this._y1 = +e);
  },
  quadraticCurveTo: function(t, e, n, r) {
    this._ += "Q" + +t + "," + +e + "," + (this._x1 = +n) + "," + (this._y1 = +r);
  },
  bezierCurveTo: function(t, e, n, r, i, o) {
    this._ += "C" + +t + "," + +e + "," + +n + "," + +r + "," + (this._x1 = +i) + "," + (this._y1 = +o);
  },
  arcTo: function(t, e, n, r, i) {
    t = +t, e = +e, n = +n, r = +r, i = +i;
    var o = this._x1, s = this._y1, u = n - t, l = r - e, f = o - t, c = s - e, p = f * f + c * c;
    if (i < 0)
      throw new Error("negative radius: " + i);
    if (this._x1 === null)
      this._ += "M" + (this._x1 = t) + "," + (this._y1 = e);
    else if (p > J)
      if (!(Math.abs(c * u - l * f) > J) || !i)
        this._ += "L" + (this._x1 = t) + "," + (this._y1 = e);
      else {
        var g = n - o, _ = r - s, E = u * u + l * l, C = g * g + _ * _, F = Math.sqrt(E), D = Math.sqrt(p), R = i * Math.tan((le - Math.acos((E + p - C) / (2 * F * D))) / 2), H = R / D, j = R / F;
        Math.abs(H - 1) > J && (this._ += "L" + (t + H * f) + "," + (e + H * c)), this._ += "A" + i + "," + i + ",0,0," + +(c * g > f * _) + "," + (this._x1 = t + j * u) + "," + (this._y1 = e + j * l);
      }
  },
  arc: function(t, e, n, r, i, o) {
    t = +t, e = +e, n = +n, o = !!o;
    var s = n * Math.cos(r), u = n * Math.sin(r), l = t + s, f = e + u, c = 1 ^ o, p = o ? r - i : i - r;
    if (n < 0)
      throw new Error("negative radius: " + n);
    this._x1 === null ? this._ += "M" + l + "," + f : (Math.abs(this._x1 - l) > J || Math.abs(this._y1 - f) > J) && (this._ += "L" + l + "," + f), n && (p < 0 && (p = p % ce + ce), p > ls ? this._ += "A" + n + "," + n + ",0,1," + c + "," + (t - s) + "," + (e - u) + "A" + n + "," + n + ",0,1," + c + "," + (this._x1 = l) + "," + (this._y1 = f) : p > J && (this._ += "A" + n + "," + n + ",0," + +(p >= le) + "," + c + "," + (this._x1 = t + n * Math.cos(i)) + "," + (this._y1 = e + n * Math.sin(i))));
  },
  rect: function(t, e, n, r) {
    this._ += "M" + (this._x0 = this._x1 = +t) + "," + (this._y0 = this._y1 = +e) + "h" + +n + "v" + +r + "h" + -n + "Z";
  },
  toString: function() {
    return this._;
  }
};
function nt(t) {
  return function() {
    return t;
  };
}
function cs(t) {
  return typeof t == "object" && "length" in t ? t : Array.from(t);
}
function Wn(t) {
  this._context = t;
}
Wn.prototype = {
  areaStart: function() {
    this._line = 0;
  },
  areaEnd: function() {
    this._line = NaN;
  },
  lineStart: function() {
    this._point = 0;
  },
  lineEnd: function() {
    (this._line || this._line !== 0 && this._point === 1) && this._context.closePath(), this._line = 1 - this._line;
  },
  point: function(t, e) {
    switch (t = +t, e = +e, this._point) {
      case 0:
        this._point = 1, this._line ? this._context.lineTo(t, e) : this._context.moveTo(t, e);
        break;
      case 1:
        this._point = 2;
      default:
        this._context.lineTo(t, e);
        break;
    }
  }
};
function jn(t) {
  return new Wn(t);
}
function fs(t) {
  return t[0];
}
function hs(t) {
  return t[1];
}
function ps(t, e) {
  var n = nt(!0), r = null, i = jn, o = null;
  t = typeof t == "function" ? t : t === void 0 ? fs : nt(t), e = typeof e == "function" ? e : e === void 0 ? hs : nt(e);
  function s(u) {
    var l, f = (u = cs(u)).length, c, p = !1, g;
    for (r == null && (o = i(g = Un())), l = 0; l <= f; ++l)
      !(l < f && n(c = u[l], l, u)) === p && ((p = !p) ? o.lineStart() : o.lineEnd()), p && o.point(+t(c, l, u), +e(c, l, u));
    if (g)
      return o = null, g + "" || null;
  }
  return s.x = function(u) {
    return arguments.length ? (t = typeof u == "function" ? u : nt(+u), s) : t;
  }, s.y = function(u) {
    return arguments.length ? (e = typeof u == "function" ? u : nt(+u), s) : e;
  }, s.defined = function(u) {
    return arguments.length ? (n = typeof u == "function" ? u : nt(!!u), s) : n;
  }, s.curve = function(u) {
    return arguments.length ? (i = u, r != null && (o = i(r)), s) : i;
  }, s.context = function(u) {
    return arguments.length ? (u == null ? r = o = null : o = i(r = u), s) : r;
  }, s;
}
function rt(t, e, n) {
  this.k = t, this.x = e, this.y = n;
}
rt.prototype = {
  constructor: rt,
  scale: function(t) {
    return t === 1 ? this : new rt(this.k * t, this.x, this.y);
  },
  translate: function(t, e) {
    return t === 0 & e === 0 ? this : new rt(this.k, this.x + this.k * t, this.y + this.k * e);
  },
  apply: function(t) {
    return [t[0] * this.k + this.x, t[1] * this.k + this.y];
  },
  applyX: function(t) {
    return t * this.k + this.x;
  },
  applyY: function(t) {
    return t * this.k + this.y;
  },
  invert: function(t) {
    return [(t[0] - this.x) / this.k, (t[1] - this.y) / this.k];
  },
  invertX: function(t) {
    return (t - this.x) / this.k;
  },
  invertY: function(t) {
    return (t - this.y) / this.k;
  },
  rescaleX: function(t) {
    return t.copy().domain(t.range().map(this.invertX, this).map(t.invert, t));
  },
  rescaleY: function(t) {
    return t.copy().domain(t.range().map(this.invertY, this).map(t.invert, t));
  },
  toString: function() {
    return "translate(" + this.x + "," + this.y + ") scale(" + this.k + ")";
  }
};
new rt(1, 0, 0);
rt.prototype;
const ds = ({
  edgeLength: t,
  metric: e
}) => {
  const n = e.value / e.max;
  return `${t * n * 6} ${t * (1 - n) * 6}`;
}, gs = Math.abs(Math.cos(2 * Math.PI / 3)), _s = Math.abs(Math.sin(2 * Math.PI / 3)), ys = ({
  edgeLength: t,
  center: e = [0, 0]
}) => {
  const n = _s * t, r = gs * t;
  return [[0, t / 2 + r], [-n, t / 2], [-n, -t / 2], [0, -(t / 2 + r)], [n, -t / 2], [n, t / 2], [0, t / 2 + r]].map(([i, o]) => [i + e[0], o + e[1]]);
}, vs = ps().curve(jn).x((t) => t[0]).y((t) => t[1]), Ss = ({
  strokeWidth: t = 5,
  metrics: e = [],
  title: n,
  ...r
}) => {
  const i = Ar(null), [o, s] = vn(0), {
    theme: u
  } = wn();
  return ee(() => {
    const l = Ke(i.current);
    l.selectAll("*").remove(), l.attr("viewBox", "0 0 100 100");
  }, []), ee(() => {
    const l = Ke(i.current);
    e.forEach((f, c) => {
      const p = `path-${c}`, g = c >= o ? l.append("path") : l.select(`#${p}`), _ = 25 + c * t * 1.25;
      g.attr("id", p).attr("stroke", u.colors.visualization.palettes.default[c]).attr("fill", "none").attr("stroke-width", t).attr("d", vs(ys({
        edgeLength: _,
        center: [50, 50]
      }))).transition().duration(1e3).ease(is).attr("stroke-dasharray", ds({
        edgeLength: _,
        metric: f
      }));
    }), l.selectAll("path").filter((f, c) => c > e.length - 1).remove(), s(e.length);
  }, [e, t]), /* @__PURE__ */ A("svg", {
    ref: i,
    ...r
  });
}, pn = {
  h1: "h1",
  h2: "h2",
  h3: "h3",
  h4: "h4",
  h5: "h5",
  p: "p",
  small: "h6"
}, dn = "h1", gn = ({
  level: t = dn,
  ...e
}) => (console.log(pn[t]), kr(pn[t] || dn, {
  ...e
}));
const Cs = ({
  value: t,
  level: e = "h4",
  variant: n = "primary",
  label: r,
  color: i
}) => /* @__PURE__ */ Vt(pe, {
  empty: !0,
  direction: "vertical",
  align: "center",
  justify: "center",
  children: [/* @__PURE__ */ A(gn, {
    className: st("pluto-value__text", n && `pluto-value__text--${n}`),
    level: e,
    children: t
  }), r && /* @__PURE__ */ A(gn, {
    className: "pluto-value__label",
    level: "small",
    children: r
  })]
});
export {
  ms as Button,
  Ts as Header,
  Ss as HexagonBar,
  xs as Input,
  pe as Space,
  Cs as Statistic,
  Es as ThemeProvider,
  Rs as ThemeSwitch,
  qr as applyThemeAsCssVars,
  bs as aryaDark,
  Yr as aryaLight,
  wn as useThemeContext
};
//# sourceMappingURL=pluto.es.js.map
