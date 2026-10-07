Local copy of github.com/lib-x/edgetts v0.4.0 (MIT; see LICENSE).

Changes: cancel WebSocket dialing and close established connections on context cancellation; drain producer sends on early exit; propagate mid-stream read errors; limit WebSocket frames to 1 MiB; close/cancel the SDK stream in both directions. Production Go sources retained; upstream tests and demo omitted. Remove this replacement when upstream provides equivalent fixes.

Update the Chromium compatibility version to 143.0.3650.75, matching https://github.com/rany2/edge-tts/blob/master/src/edge_tts/constants.py at implementation time; include HTTP status in handshake failures.
