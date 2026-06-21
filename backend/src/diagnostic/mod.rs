//! Diagnostic module for build process error handling

mod errors;

pub use errors::{
    BuildSummary,
    DiagnosticBuildError,
    DiagnosticBuilder,
    DiagnosticResult,
};