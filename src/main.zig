// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

const std = @import("std");
const core = @import("core/root.zig");
const network = @import("network/root.zig");
const pfte = @import("pfte/root.zig");
const cli = @import("cli/root.zig");

/// Entry point.
/// Keep this lightweight, the heavy lifting happens in the engine modules.
pub fn main() !void {
    // Using GPA for now. If memory fragmentation becomes an issue during heavy
    // transfer loads, we might switch to a fixed buffer or Arena for the Engine.
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    const stdout = std.io.getStdOut().writer();

    // Just to know we are alive
    try stdout.print("FileRipper v0.1.0 - PFTE Init\n", .{});

    if (args.len < 2) {
        try printUsage(stdout);
        return;
    }

    // Manual arg parsing is fine for this stage.
    // We expect the UI (Flutter) to be strict, people using CLI might make mistakes.
    const command = args[1];

    if (std.mem.eql(u8, command, "start-server")) {
        // This is the daemon mode for the UI.
        try stdout.print(">> Starting PFTE Server loop...\n", .{});
        try cli.handleServerCommand();
    } else if (std.mem.eql(u8, command, "transfer")) {
        // One-off CLI command.
        try stdout.print(">> CLI Transfer mode engaged.\n", .{});
        
        // Test init of the engine just to see if it crashes
        var engine = pfte.Engine.init(allocator);
        defer engine.deinit();
        
        // Mock session
        var session = try network.SftpSession.init(allocator, "localhost");
        defer session.close();

        try engine.startTransfer(&session);

    } else {
        try stdout.print("Invalid command: {s}\n", .{command});
        try printUsage(stdout);
    }
}

fn printUsage(writer: anytype) !void {
    try writer.print(
        \\Usage: fileripper [command]
        \\
        \\Commands:
        \\  start-server   Daemon mode (API for Flutter UI)
        \\  transfer       CLI mode (Debug/Scripts)
        \\
    , .{});
}