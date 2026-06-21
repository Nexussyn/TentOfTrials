//! Diagnostic build error handling module
//!
//! Provides robust error handling for the diagnostic build process
//! with graceful failure modes and diagnostic logging.

use std::fmt;
use std::io;
use std::error::Error;

/// Error types for diagnostic build process
#[derive(Debug)]
pub enum DiagnosticBuildError {
    /// Configuration error during build setup
    ConfigError(String),
    /// File system operation failed
    IoError(io::Error),
    /// Build step failed with details
    BuildStepFailed { step: String, reason: String },
    /// Timeout during build operation
    Timeout { operation: String, duration_secs: u64 },
    /// Resource exhaustion (memory, disk, etc.)
    ResourceExhausted(String),
    /// Validation failed for build artifacts
    ValidationFailed(String),
}

impl fmt::Display for DiagnosticBuildError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::ConfigError(msg) => write!(f, "Configuration error: {}", msg),
            Self::IoError(e) => write!(f, "IO error: {}", e),
            Self::BuildStepFailed { step, reason } => {
                write!(f, "Build step '{}' failed: {}", step, reason)
            }
            Self::Timeout { operation, duration_secs } => {
                write!(f, "Operation '{}' timed out after {}s", operation, duration_secs)
            }
            Self::ResourceExhausted(resource) => {
                write!(f, "Resource exhausted: {}", resource)
            }
            Self::ValidationFailed(msg) => write!(f, "Validation failed: {}", msg),
        }
    }
}

impl Error for DiagnosticBuildError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::IoError(e) => Some(e),
            _ => None,
        }
    }
}

impl From<io::Error> for DiagnosticBuildError {
    fn from(err: io::Error) -> Self {
        Self::IoError(err)
    }
}

/// Result type alias for diagnostic build operations
pub type DiagnosticResult<T> = Result<T, DiagnosticBuildError>;

/// Diagnostic builder with error handling and logging
pub struct DiagnosticBuilder {
    logs: Vec<String>,
    errors: Vec<DiagnosticBuildError>,
    continue_on_error: bool,
}

impl DiagnosticBuilder {
    /// Create a new diagnostic builder
    pub fn new() -> Self {
        Self {
            logs: Vec::new(),
            errors: Vec::new(),
            continue_on_error: false,
        }
    }

    /// Set whether to continue on non-fatal errors
    pub fn continue_on_error(mut self, value: bool) -> Self {
        self.continue_on_error = value;
        self
    }

    /// Log a diagnostic message
    pub fn log(&mut self, message: &str) {
        let timestamp = chrono::Utc::now().format("%Y-%m-%dT%H:%M:%SZ");
        let log_entry = format!("[{}] {}", timestamp, message);
        self.logs.push(log_entry);
    }

    /// Log an error and optionally continue
    pub fn handle_error(&mut self, error: DiagnosticBuildError) -> DiagnosticResult<()> {
        self.log(&format!("ERROR: {}", error));
        
        if self.continue_on_error {
            self.errors.push(error);
            Ok(())
        } else {
            Err(error)
        }
    }

    /// Execute a build step with error handling
    pub fn execute_step<F, T>(&mut self, step_name: &str, operation: F) -> DiagnosticResult<T>
    where
        F: FnOnce() -> DiagnosticResult<T>,
    {
        self.log(&format!("Starting step: {}", step_name));
        
        match operation() {
            Ok(result) => {
                self.log(&format!("Completed step: {}", step_name));
                Ok(result)
            }
            Err(e) => {
                let wrapped = DiagnosticBuildError::BuildStepFailed {
                    step: step_name.to_string(),
                    reason: e.to_string(),
                };
                self.handle_error(wrapped).and_then(|_| Err(e))
            }
        }
    }

    /// Get all diagnostic logs
    pub fn get_logs(&self) -> &[String] {
        &self.logs
    }

    /// Get all collected errors
    pub fn get_errors(&self) -> &[DiagnosticBuildError] {
        &self.errors
    }

    /// Check if build had any errors
    pub fn has_errors(&self) -> bool {
        !self.errors.is_empty()
    }

    /// Finalize and return build summary
    pub fn finalize(self) -> BuildSummary {
        BuildSummary {
            logs: self.logs,
            error_count: self.errors.len(),
            success: self.errors.is_empty(),
        }
    }
}

impl Default for DiagnosticBuilder {
    fn default() -> Self {
        Self::new()
    }
}

/// Summary of a diagnostic build run
#[derive(Debug)]
pub struct BuildSummary {
    pub logs: Vec<String>,
    pub error_count: usize,
    pub success: bool,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_diagnostic_error_display() {
        let err = DiagnosticBuildError::ConfigError("invalid path".to_string());
        assert!(err.to_string().contains("invalid path"));

        let err = DiagnosticBuildError::BuildStepFailed {
            step: "compile".to_string(),
            reason: "syntax error".to_string(),
        };
        assert!(err.to_string().contains("compile"));
    }

    #[test]
    fn test_builder_logging() {
        let mut builder = DiagnosticBuilder::new();
        builder.log("Test message");
        
        assert_eq!(builder.get_logs().len(), 1);
        assert!(builder.get_logs()[0].contains("Test message"));
    }

    #[test]
    fn test_builder_successful_step() {
        let mut builder = DiagnosticBuilder::new();
        let result = builder.execute_step("test_step", || Ok(42));
        
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 42);
        assert!(!builder.has_errors());
    }

    #[test]
    fn test_builder_failed_step() {
        let mut builder = DiagnosticBuilder::new();
        let result: DiagnosticResult<()> = builder.execute_step("failing_step", || {
            Err(DiagnosticBuildError::ConfigError("failed".to_string()))
        });
        
        assert!(result.is_err());
    }

    #[test]
    fn test_builder_continue_on_error() {
        let mut builder = DiagnosticBuilder::new().continue_on_error(true);
        
        let err = DiagnosticBuildError::ConfigError("non-fatal".to_string());
        let result = builder.handle_error(err);
        
        assert!(result.is_ok());
        assert!(builder.has_errors());
        assert_eq!(builder.get_errors().len(), 1);
    }

    #[test]
    fn test_build_summary() {
        let mut builder = DiagnosticBuilder::new();
        builder.log("Step 1");
        builder.log("Step 2");
        
        let summary = builder.finalize();
        
        assert!(summary.success);
        assert_eq!(summary.error_count, 0);
        assert_eq!(summary.logs.len(), 2);
    }
}