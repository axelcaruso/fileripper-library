// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

/// Global error definitions.
/// Try to keep this exhaustive. If the UI sees 'Unknown', we failed.
pub const RipperError = error{
    // Network layer crap
    ConnectionFailed,
    HostUnreachable,
    AuthenticationFailed, // Keys or password rejected
    Timeout,

    // PFTE Specifics
    // These happen when the pipeline gets clogged or the server chokes
    FlowOverflowError,
    PipelineStalled,
    InvalidChunkSize,

    // FS
    FileNotFound,
    PermissionDenied,
    DiskFull,

    // Sanity checks
    UnknownCommand,
    SystemResourceExhausted, // OOM or too many open handles
};