// Verifies go.withmatt.com/zon encoder output against the Zig 0.16 standard library.
// Used by zig_test.go: the test generates the .zon files this file imports,
// copies this file next to them, and runs `zig test check.zig`. It is not
// meant to be run directly from the repository root.
const std = @import("std");

const Mode = enum { @"while", debug };
const Err = union(enum) { io: []const u8, backoff: void };
const Status = enum { debug, release };
const Dep = struct { url: []const u8, hash: []const u8 };

// The struct zig_test.go marshals with reflection, then parses back.
const Marshal = struct {
    name: []const u8,
    version: []const u8,
    mode: Status,
    @"while": i64,
    user_id: i64,
    opt: ?u32,
    optnull: ?u32,
    count: i64,
    ratio: f64,
    flag: bool,
    big: u112,
    tags: []const []const u8,
    matrix: [2][2]i32,
    deps: struct { alpha: Dep, beta: Dep },
    retry: Err,
    err: Err,
    content: []const u8,
    cp: u21,
    embedded_inner: []const u8,
    proxy_v2: bool = true,
    cert_path: []const u8 = "default.pem",
};

const Sink = struct {
    nan: f64,
    inf: f64,
    ninf: f64,
    negzero: f64,
    exp: f64,
    whole: f64,
    tiny: f64,
    min: i64,
    max: i64,
    umax: u64,
    b: bool,
    opt: ?u32,
    optnull: ?u32,
    s: []const u8,
    ml: []const u8,
    cp: u21,
    arr1: [1]i32,
    arr: [3]i32,
    mode: Mode,
    kw: Mode,
    err: Err,
    retry: Err,
    @"type": struct { x: i32 },
};

test "sink round-trips through std.zon.parse" {
    @setEvalBranchQuota(100000);
    const src: [:0]const u8 = @embedFile("sink.zon");
    const v = try std.zon.parse.fromSliceAlloc(Sink, std.testing.allocator, src, null, .{});
    defer std.zon.parse.free(std.testing.allocator, v);

    try std.testing.expect(std.math.isNan(v.nan));
    try std.testing.expect(std.math.isPositiveInf(v.inf));
    try std.testing.expect(std.math.isNegativeInf(v.ninf));
    try std.testing.expect(v.negzero == 0 and std.math.signbit(v.negzero));
    try std.testing.expectEqual(@as(f64, 1e+300), v.exp);
    try std.testing.expectEqual(@as(f64, 5), v.whole);
    try std.testing.expectEqual(@as(f64, 0.1), v.tiny);
    try std.testing.expect(v.min == std.math.minInt(i64));
    try std.testing.expect(v.max == std.math.maxInt(i64));
    try std.testing.expect(v.umax == std.math.maxInt(u64));
    try std.testing.expect(v.b);
    try std.testing.expectEqual(@as(?u32, 7), v.opt);
    try std.testing.expectEqual(@as(?u32, null), v.optnull);
    try std.testing.expectEqualStrings("é \\ hi \x07 \" derp '", v.s);
    try std.testing.expectEqualStrings("a\nb", v.ml);
    try std.testing.expectEqual(@as(u21, 0x26a1), v.cp);
    try std.testing.expectEqual([1]i32{1}, v.arr1);
    try std.testing.expectEqual([3]i32{ 1, 2, 3 }, v.arr);
    try std.testing.expectEqual(Mode.debug, v.mode);
    try std.testing.expectEqual(Mode.@"while", v.kw);
    try std.testing.expectEqualStrings("EOF", v.err.io);
    try std.testing.expect(v.retry == .backoff);
    try std.testing.expectEqual(@as(i32, 1), v.@"type".x);
}

test "marshal round-trips through std.zon.parse" {
    @setEvalBranchQuota(100000);
    // An arena, not testing.allocator + free: fields filled from Zig defaults
    // (cert_path) point at static memory, which std.zon.parse.free cannot free.
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const src: [:0]const u8 = @embedFile("marshal.zon");
    const v = try std.zon.parse.fromSliceAlloc(Marshal, arena.allocator(), src, null, .{});

    try std.testing.expectEqualStrings("example", v.name);
    try std.testing.expectEqualStrings("0.1.0", v.version);
    try std.testing.expectEqual(Status.debug, v.mode);
    try std.testing.expectEqual(@as(i64, 5), v.@"while");
    try std.testing.expectEqual(@as(i64, 42), v.user_id);
    try std.testing.expectEqual(@as(?u32, 7), v.opt);
    try std.testing.expectEqual(@as(?u32, null), v.optnull);
    try std.testing.expectEqual(@as(i64, 1000000007), v.count);
    try std.testing.expectEqual(@as(f64, 0.1), v.ratio);
    try std.testing.expect(v.flag);
    try std.testing.expect(v.big == (@as(u112, 1) << 100));
    try std.testing.expectEqual(@as(usize, 3), v.tags.len);
    try std.testing.expectEqualStrings("a", v.tags[0]);
    try std.testing.expectEqualStrings("c", v.tags[2]);
    try std.testing.expectEqual([2][2]i32{ .{ 1, 2 }, .{ 3, 4 } }, v.matrix);
    try std.testing.expectEqualStrings("https://u1", v.deps.alpha.url);
    try std.testing.expectEqualStrings("h2", v.deps.beta.hash);
    try std.testing.expect(v.retry == .backoff);
    try std.testing.expectEqualStrings("EOF", v.err.io);
    try std.testing.expectEqualStrings("first\nsecond", v.content);
    try std.testing.expectEqual(@as(u21, 0x26a1), v.cp);
    try std.testing.expectEqualStrings("deep", v.embedded_inner);
    try std.testing.expect(v.proxy_v2); // present: omitempty kept true
    try std.testing.expectEqualStrings("default.pem", v.cert_path); // omitted: default filled
}

comptime {
    // u256-scale literals round-trip exactly through comptime_int; std.zon.parse
    // itself cannot compile a u256 field (its f128 range check cannot represent
    // maxInt(u256)), so wide integers are verified here instead.
    const big = @import("big.zon");
    if (big.big != (@as(u256, 1) << 255)) @compileError("big integer mismatch");

    _ = @import("fmt_wrapped.zon");
    _ = @import("fmt_compact.zon");
    _ = @import("fmt_deep.zon");
    _ = @import("ml_root.zon");
}
