"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SegmentPayloadSchema = void 0;
const zod_1 = require("zod");
const telem_1 = require("../telem");
exports.SegmentPayloadSchema = zod_1.z.object({
    channelKey: zod_1.z.string(),
    start: zod_1.z.number().transform((n) => new telem_1.TimeStamp(n)),
    data: zod_1.z.string().transform((s) => new Uint8Array(atob(s)
        .split('')
        .map((c) => c.charCodeAt(0)))),
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGF5bG9hZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uLy4uL3NyYy9saWIvc2VnbWVudC9wYXlsb2FkLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7OztBQUFBLDZCQUF3QjtBQUV4QixvQ0FBcUM7QUFFeEIsUUFBQSxvQkFBb0IsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQzNDLFVBQVUsRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFO0lBQ3RCLEtBQUssRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsU0FBUyxDQUFDLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FBQyxJQUFJLGlCQUFTLENBQUMsQ0FBQyxDQUFDLENBQUM7SUFDcEQsSUFBSSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxTQUFTLENBQ3hCLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FDSixJQUFJLFVBQVUsQ0FDWixJQUFJLENBQUMsQ0FBQyxDQUFDO1NBQ0osS0FBSyxDQUFDLEVBQUUsQ0FBQztTQUNULEdBQUcsQ0FBQyxDQUFDLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDLFVBQVUsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUMvQixDQUNKO0NBQ0YsQ0FBQyxDQUFDIn0=