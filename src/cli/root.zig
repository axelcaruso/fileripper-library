// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

const std = @import("std");

/// Handles the command line logic.
/// Keeps main.zig clean.
pub fn handleTransferCommand(args: [][:0]u8) !void {
    _ = args;
    // We will move the arg parsing logic here.
    // std.debug.print("CLI Handler: Processing {d} args\n", .{args.len});
}

pub fn handleServerCommand() !void {
    // This will act as the daemon entry point.
}