// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

const std = @import("std");
const network = @import("../network/root.zig");

/// The magic number.
/// We fire this many requests before checking if the server is crying.
const BATCH_SIZE_BOOST: usize = 64;
const BATCH_SIZE_CONSERVATIVE: usize = 2;

pub const Engine = struct {
    allocator: std.mem.Allocator,
    queue: std.ArrayList([]const u8), // Simple path queue for now
    mode: TransferMode,

    pub const TransferMode = enum {
        Boost,       // The "Flow Overflow" mode
        Conservative // The "Please don't ban me" mode
    };

    pub fn init(allocator: std.mem.Allocator) Engine {
        return Engine{
            .allocator = allocator,
            .queue = std.ArrayList([]const u8).init(allocator),
            .mode = .Boost, // Default to speed
        };
    }

    pub fn deinit(self: *Engine) void {
        self.queue.deinit();
    }

    /// This is where we will implement the parallel logic.
    pub fn startTransfer(self: *Engine, session: *network.SftpSession) !void {
        _ = session;
        // Logic flow:
        // 1. Fill the pipe (64 requests).
        // 2. Wait for ANY response (io_uring or polling).
        // 3. Refill immediately.
        
        // Use debug print for now so we know it's being called
        std.debug.print(">> PFTE Engine started in {s} mode. Batch size: {d}\n", .{
            @tagName(self.mode),
            if (self.mode == .Boost) BATCH_SIZE_BOOST else BATCH_SIZE_CONSERVATIVE
        });
    }
};