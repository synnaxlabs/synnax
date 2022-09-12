"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ENCODERS = exports.JSONEncoderDecoder = exports.MsgPackEncoderDecoder = void 0;
const msgpack_1 = require("@msgpack/msgpack");
const caseconv_1 = require("./caseconv");
class MsgPackEncoderDecoder {
    constructor() {
        this.contentType = 'application/msgpack';
    }
    encode(payload) {
        return (0, msgpack_1.encode)(payload);
    }
    decode(data) {
        return (0, msgpack_1.decode)(data);
    }
}
exports.MsgPackEncoderDecoder = MsgPackEncoderDecoder;
class JSONEncoderDecoder {
    constructor() {
        this.contentType = 'application/json';
    }
    encode(payload) {
        return new TextEncoder().encode(JSON.stringify((0, caseconv_1.snakeKeys)(payload)));
    }
    decode(data) {
        return (0, caseconv_1.camelKeys)(JSON.parse(new TextDecoder().decode(data)));
    }
}
exports.JSONEncoderDecoder = JSONEncoderDecoder;
exports.ENCODERS = [
    new MsgPackEncoderDecoder(),
    new JSONEncoderDecoder(),
];
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5jb2Rlci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvZW5jb2Rlci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSw4Q0FBa0Q7QUFFbEQseUNBQWtEO0FBU2xELE1BQWEscUJBQXFCO0lBQWxDO1FBQ0UsZ0JBQVcsR0FBRyxxQkFBcUIsQ0FBQztJQVN0QyxDQUFDO0lBUEMsTUFBTSxDQUFDLE9BQWdCO1FBQ3JCLE9BQU8sSUFBQSxnQkFBTSxFQUFDLE9BQU8sQ0FBQyxDQUFDO0lBQ3pCLENBQUM7SUFFRCxNQUFNLENBQUksSUFBZ0I7UUFDeEIsT0FBTyxJQUFBLGdCQUFNLEVBQUMsSUFBSSxDQUFpQixDQUFDO0lBQ3RDLENBQUM7Q0FDRjtBQVZELHNEQVVDO0FBRUQsTUFBYSxrQkFBa0I7SUFBL0I7UUFDRSxnQkFBVyxHQUFHLGtCQUFrQixDQUFDO0lBV25DLENBQUM7SUFUQyxNQUFNLENBQUMsT0FBZ0I7UUFDckIsT0FBTyxJQUFJLFdBQVcsRUFBRSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLElBQUEsb0JBQVMsRUFBQyxPQUFPLENBQUMsQ0FBQyxDQUFDLENBQUM7SUFDdEUsQ0FBQztJQUVELE1BQU0sQ0FBSSxJQUFnQjtRQUN4QixPQUFPLElBQUEsb0JBQVMsRUFDZCxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksV0FBVyxFQUFFLENBQUMsTUFBTSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQzNCLENBQUM7SUFDcEIsQ0FBQztDQUNGO0FBWkQsZ0RBWUM7QUFFWSxRQUFBLFFBQVEsR0FBcUI7SUFDeEMsSUFBSSxxQkFBcUIsRUFBRTtJQUMzQixJQUFJLGtCQUFrQixFQUFFO0NBQ3pCLENBQUMifQ==