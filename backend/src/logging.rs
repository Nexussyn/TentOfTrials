# Fix for Issue #2: [$15 BOUNTY] [Rust] Log selected backend log format at startup

// backend/src/logging.rs
use std::env;
use tracing::{info, error};
use tracing_subscriber::fmt;
use tracing_subscriber::EnvFilter;

/// Represents the available log format options
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LogFormat {
    Text,
    Json,
}

impl LogFormat {
    /// Returns the string representation of the log format
    pub fn as_str(&self) -> &'static str {
        match self {
            LogFormat::Text => "text",
            LogFormat::Json => "json",
        }
    }
}

impl std::fmt::Display for LogFormat {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Error type for invalid log format values
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InvalidLogFormatError {
    pub value: String,
}

impl std::fmt::Display for InvalidLogFormatError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "invalid log format '{}': expected 'text' or 'json'", self.value)
    }
}

impl std::error::Error for InvalidLogFormatError {}

/// Parses the log format from a string value
pub fn parse_log_format(value: &str) -> Result<LogFormat, InvalidLogFormatError> {
    match value.to_lowercase().as_str() {
        "text" => Ok(LogFormat::Text),
        "json" => Ok(LogFormat::Json),
        _ => Err(InvalidLogFormatError { value: value.to_string() }),
    }
}

/// Determines the log format from the TOT_LOG_FORMAT environment variable
/// Returns LogFormat::Text as the default if the variable is not set
/// Returns an error if the variable contains an invalid value
pub fn get_log_format_from_env() -> Result<LogFormat, InvalidLogFormatError> {
    match env::var("TOT_LOG_FORMAT") {
        Ok(value) => parse_log_format(&value),
        Err(env::VarError::NotPresent) => Ok(LogFormat::Text),
        Err(env::VarError::NotUnicode(_)) => Err(InvalidLogFormatError {
            value: "<invalid unicode>".to_string(),
        }),
    }
}

/// Initializes the logging subsystem based on the TOT_LOG_FORMAT environment variable
/// Logs the selected format at startup for operator confirmation
pub fn init_logging() -> Result<LogFormat, InvalidLogFormatError> {
    let format = get_log_format_from_env()?;
    
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    match format {
        LogFormat::Text => {
            let subscriber = fmt::Subscriber::builder()
                .with_env_filter(env_filter)
                .with_target(true)
                .with_thread_ids(false)
                .with_file(false)
                .with_line_number(false)
                .finish();
            tracing::subscriber::set_global_default(subscriber)
                .expect("failed to set global default subscriber");
        }
        LogFormat::Json => {
            let subscriber = fmt::Subscriber::builder()
                .with_env_filter(env_filter)
                .with_target(true)
                .json()
                .finish();
            tracing::subscriber::set_global_default(subscriber)
                .expect("failed to set global default subscriber");
        }
    }

    // Log the selected format at startup for operator confirmation
    info!(log_format = format.as_str(), "logging subsystem initialized");

    Ok(format)
}

/// Returns the startup log format value for display/testing purposes
/// This is a helper function that can be used to get the format value
/// without fully initializing the logging system
pub fn get_startup_format_display() -> Result<String, InvalidLogFormatError> {
    let format = get_log_format_from_env()?;
    Ok(format.as_str().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::env;

    fn with_env_var<F, R>(key: &str, value: Option<&str>, f: F) -> R
    where
        F: FnOnce() -> R,
    {
        let original = env::var(key).ok();
        
        match value {
            Some(v) => env::set_var(key, v),
            None => env::remove_var(key),
        }
        
        let result = f();
        
        match original {
            Some(v) => env::set_var(key, v),
            None => env::remove_var(key),
        }
        
        result
    }

    #[test]
    fn test_parse_log_format_text() {
        assert_eq!(parse_log_format("text").unwrap(), LogFormat::Text);
        assert_eq!(parse_log_format("TEXT").unwrap(), LogFormat::Text);
        assert_eq!(parse_log_format("Text").unwrap(), LogFormat::Text);
    }

    #[test]
    fn test_parse_log_format_json() {
        assert_eq!(parse_log_format("json").unwrap(), LogFormat::Json);
        assert_eq!(parse_log_format("JSON").unwrap(), LogFormat::Json);
        assert_eq!(parse_log_format("Json").unwrap(), LogFormat::Json);
    }

    #[test]
    fn test_parse_log_format_invalid() {
        let err = parse_log_format("invalid").unwrap_err();
        assert_eq!(err.value, "invalid");
        
        let err = parse_log_format("xml").unwrap_err();
        assert_eq!(err.value, "xml");
        
        let err = parse_log_format("").unwrap_err();
        assert_eq!(err.value, "");
    }

    #[test]
    fn test_log_format_as_str() {
        assert_eq!(LogFormat::Text.as_str(), "text");
        assert_eq!(LogFormat::Json.as_str(), "json");
    }

    #[test]
    fn test_log_format_display() {
        assert_eq!(format!("{}", LogFormat::Text), "text");
        assert_eq!(format!("{}", LogFormat::Json), "json");
    }

    #[test]
    fn test_get_log_format_from_env_default() {
        with_env_var("TOT_LOG_FORMAT", None, || {
            let format = get_log_format_from_env().unwrap();
            assert_eq!(format, LogFormat::Text);
        });
    }

    #[test]
    fn test_get_log_format_from_env_text() {
        with_env_var("TOT_LOG_FORMAT", Some("text"), || {
            let format = get_log_format_from_env().unwrap();
            assert_eq!(format, LogFormat::Text);
        });
    }

    #[test]
    fn test_get_log_format_from_env_json() {
        with_env_var("TOT_LOG_FORMAT", Some("json"), || {
            let format = get_log_format_from_env().unwrap();
            assert_eq!(format, LogFormat::Json);
        });
    }

    #[test]
    fn test_get_log_format_from_env_invalid() {
        with_env_var("TOT_LOG_FORMAT", Some("invalid"), || {
            let err = get_log_format_from_env().unwrap_err();
            assert_eq!(err.value, "invalid");
        });
    }

    #[test]
    fn test_get_startup_format_display_text() {
        with_env_var("TOT_LOG_FORMAT", Some("text"), || {
            let display = get_startup_format_display().unwrap();
            assert_eq!(display, "text");
        });
    }

    #[test]
    fn test_get_startup_format_display_json() {
        with_env_var("TOT_LOG_FORMAT", Some("json"), || {
            let display = get_startup_format_display().unwrap();
            assert_eq!(display, "json");
        });
    }

    #[test]
    fn test_get_startup_format_display_default() {
        with_env_var("TOT_LOG_FORMAT", None, || {
            let display = get_startup_format_display().unwrap();
            assert_eq!(display, "text");
        });
    }

    #[test]
    fn test_invalid_log_format_error_display() {
        let err = InvalidLogFormatError { value: "badvalue".to_string() };
        assert_eq!(
            format!("{}", err),
            "invalid log format 'badvalue': expected 'text' or 'json'"
        );
    }
}