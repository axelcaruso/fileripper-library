// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

const std = @import("std");

pub fn build(b: *std.Build) void {
    // Let the user decide the target. Cross-compilation is Zig's superpower anyway,
    // so no need to hardcode OS specifics here yet.
    const target = b.standardTargetOptions(.{});

    // We stick to standard optimizations.
    // ReleaseFast for benchmarks, Debug for... well, debugging.
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "fileripper",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });

    b.installArtifact(exe);

    // Run step logic.
    // This is mostly for local dev loops.
    const run_cmd = b.addRunArtifact(exe);
    run_cmd.step.dependOn(b.getInstallStep());

    if (b.args) |args| {
        run_cmd.addArgs(args);
    }

    const run_step = b.step("run", "Run the app");
    run_step.dependOn(&run_cmd.step);
}