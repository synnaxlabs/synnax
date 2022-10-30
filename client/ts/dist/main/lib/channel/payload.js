"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.channelPayloadSchema = void 0;
const zod_1 = require("zod");
const telem_1 = require("../telem");
exports.channelPayloadSchema = zod_1.z.object({
    rate: zod_1.z.number().transform((n) => new telem_1.Rate(n)),
    dataType: zod_1.z.string().transform((s) => new telem_1.DataType(s)),
    key: zod_1.z.string().default('').optional(),
    name: zod_1.z.string().default('').optional(),
    nodeId: zod_1.z.number().default(0).optional(),
    density: zod_1.z
        .number()
        .default(0)
        .transform((n) => new telem_1.Density(n))
        .optional(),
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGF5bG9hZC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uLy4uL3NyYy9saWIvY2hhbm5lbC9wYXlsb2FkLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7OztBQUFBLDZCQUF3QjtBQUV4QixvQ0FBbUQ7QUFFdEMsUUFBQSxvQkFBb0IsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQzNDLElBQUksRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsU0FBUyxDQUFDLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FBQyxJQUFJLFlBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztJQUM5QyxRQUFRLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUMsRUFBRSxFQUFFLENBQUMsSUFBSSxnQkFBUSxDQUFDLENBQUMsQ0FBQyxDQUFDO0lBQ3RELEdBQUcsRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFLENBQUMsT0FBTyxDQUFDLEVBQUUsQ0FBQyxDQUFDLFFBQVEsRUFBRTtJQUN0QyxJQUFJLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLE9BQU8sQ0FBQyxFQUFFLENBQUMsQ0FBQyxRQUFRLEVBQUU7SUFDdkMsTUFBTSxFQUFFLE9BQUMsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxPQUFPLENBQUMsQ0FBQyxDQUFDLENBQUMsUUFBUSxFQUFFO0lBQ3hDLE9BQU8sRUFBRSxPQUFDO1NBQ1AsTUFBTSxFQUFFO1NBQ1IsT0FBTyxDQUFDLENBQUMsQ0FBQztTQUNWLFNBQVMsQ0FBQyxDQUFDLENBQUMsRUFBRSxFQUFFLENBQUMsSUFBSSxlQUFPLENBQUMsQ0FBQyxDQUFDLENBQUM7U0FDaEMsUUFBUSxFQUFFO0NBQ2QsQ0FBQyxDQUFDIn0=