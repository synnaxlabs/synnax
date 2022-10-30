"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ValidationError = exports.UnexpectedError = exports.RouteError = exports.QueryError = exports.ParseError = exports.GeneralError = exports.ContiguityError = exports.AuthError = exports.synnaxPropsSchema = exports.Synnax = void 0;
var client_1 = require("./lib/client");
Object.defineProperty(exports, "Synnax", { enumerable: true, get: function () { return __importDefault(client_1).default; } });
Object.defineProperty(exports, "synnaxPropsSchema", { enumerable: true, get: function () { return client_1.synnaxPropsSchema; } });
__exportStar(require("./lib/telem"), exports);
var errors_1 = require("./lib/errors");
Object.defineProperty(exports, "AuthError", { enumerable: true, get: function () { return errors_1.AuthError; } });
Object.defineProperty(exports, "ContiguityError", { enumerable: true, get: function () { return errors_1.ContiguityError; } });
Object.defineProperty(exports, "GeneralError", { enumerable: true, get: function () { return errors_1.GeneralError; } });
Object.defineProperty(exports, "ParseError", { enumerable: true, get: function () { return errors_1.ParseError; } });
Object.defineProperty(exports, "QueryError", { enumerable: true, get: function () { return errors_1.QueryError; } });
Object.defineProperty(exports, "RouteError", { enumerable: true, get: function () { return errors_1.RouteError; } });
Object.defineProperty(exports, "UnexpectedError", { enumerable: true, get: function () { return errors_1.UnexpectedError; } });
Object.defineProperty(exports, "ValidationError", { enumerable: true, get: function () { return errors_1.ValidationError; } });
__exportStar(require("./lib/channel"), exports);
__exportStar(require("./lib/ontology"), exports);
__exportStar(require("./lib/connectivity"), exports);
__exportStar(require("./lib/ontology"), exports);
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaW5kZXguanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi9zcmMvaW5kZXgudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7QUFBQSx1Q0FJc0I7QUFIcEIsaUhBQUEsT0FBTyxPQUFVO0FBQ2pCLDJHQUFBLGlCQUFpQixPQUFBO0FBR25CLDhDQUE0QjtBQUM1Qix1Q0FTc0I7QUFScEIsbUdBQUEsU0FBUyxPQUFBO0FBQ1QseUdBQUEsZUFBZSxPQUFBO0FBQ2Ysc0dBQUEsWUFBWSxPQUFBO0FBQ1osb0dBQUEsVUFBVSxPQUFBO0FBQ1Ysb0dBQUEsVUFBVSxPQUFBO0FBQ1Ysb0dBQUEsVUFBVSxPQUFBO0FBQ1YseUdBQUEsZUFBZSxPQUFBO0FBQ2YseUdBQUEsZUFBZSxPQUFBO0FBRWpCLGdEQUE4QjtBQUM5QixpREFBK0I7QUFDL0IscURBQW1DO0FBQ25DLGlEQUErQiJ9