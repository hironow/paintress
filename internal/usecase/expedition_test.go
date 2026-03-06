package usecase
// white-box-reason: usecase internals: tests unexported expedition adapter construction

// Validation tests for RunExpeditionCommand and ArchivePruneCommand have been
// moved to domain/primitives_test.go (parse-don't-validate). The usecase layer
// no longer calls Validate() — commands are always-valid by construction.
