// Package port defines context-aware interface contracts and trivial default
// implementations (null objects) for the port-adapter pattern.
// Concrete I/O implementations live in session and platform layers.
// Port resides under usecase/ to express the architectural dependency:
//
//	usecase → usecase/port → domain (+ stdlib such as context, errors)
//
// No imports of upper internal layers (cmd, usecase root, session, eventsource, platform).
package port
