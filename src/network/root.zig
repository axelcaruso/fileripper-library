// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

const std = @import("std");
const errors = @import("../core/root.zig").errors;

/// Represents a raw connection to the server.
/// Later this will hold the C pointers to the SSH session.
pub const SftpSession = struct {
    hostname: []const u8,
    port: u16,
    // ssh_session: *c.ssh_session, // We'll add the C types later
    // sftp_session: *c.sftp_session,

    pub fn init(allocator: std.mem.Allocator, host: []const u8) !SftpSession {
        _ = allocator; // Silence unused var error for now
        // TODO: Actually connect to something.
        // For now, we pretend we did it.
        return SftpSession{
            .hostname = host,
            .port = 22,
        };
    }

    pub fn connect(self: *SftpSession) !void {
        // Here be dragons (networking logic).
        // If this fails, we return a core error.
        if (self.port == 0) return errors.RipperError.ConnectionFailed;
    }

    pub fn close(self: *SftpSession) void {
        _ = self;
        // Clean up memory leaks here later.
    }
};