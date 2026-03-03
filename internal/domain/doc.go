// Package domain contains domain types, events, commands, policies, and pure
// domain logic. Functions here have no I/O and no context.Context except for
// OTel metric recording which is fire-and-forget.
// I/O operations belong in the session layer; orchestration belongs in usecase.
package domain
