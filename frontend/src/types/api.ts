# Fix for Issue #1: [$40 BOUNTY] [TypeScript] Harden API response error handling

/**
 * API Types for TentOfTrials Frontend
 * Defines structured types for API responses and errors
 */

/**
 * Standard successful API response wrapper
 */
export interface ApiResponse<T> {
  data: T;
  status: number;
  statusText: string;
  headers: Headers;
  requestId?: string;
}

/**
 * Structured API error with full context
 */
export class ApiError extends Error {
  public readonly status: number;
  public readonly statusText: string;
  public readonly requestId?: string;
  public readonly path?: string;
  public readonly details?: Record<string, unknown>;
  public readonly suggestion?: string;
  public readonly timestamp: string;
  public readonly isApiError = true;

  constructor(options: {
    message: string;
    status: number;
    statusText: string;
    requestId?: string;
    path?: string;
    details?: Record<string, unknown>;
    suggestion?: string;
  }) {
    super(options.message);
    this.name = 'ApiError';
    this.status = options.status;
    this.statusText = options.statusText;
    this.requestId = options.requestId;
    this.path = options.path;
    this.details = options.details;
    this.suggestion = options.suggestion;
    this.timestamp = new Date().toISOString();

    // Maintains proper stack trace for where error was thrown
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, ApiError);
    }
  }

  /**
   * Check if an error is an ApiError instance
   */
  static isApiError(error: unknown): error is ApiError {
    return (
      error instanceof ApiError ||
      (typeof error === 'object' &&
        error !== null &&
        'isApiError' in error &&
        (error as { isApiError: boolean }).isApiError === true)
    );
  }

  /**
   * Create a user-friendly error message
   */
  toUserMessage(): string {
    if (this.suggestion) {
      return `${this.message}. ${this.suggestion}`;
    }
    return this.message;
  }

  /**
   * Serialize error for logging
   */
  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      message: this.message,
      status: this.status,
      statusText: this.statusText,
      requestId: this.requestId,
      path: this.path,
      details: this.details,
      suggestion: this.suggestion,
      timestamp: this.timestamp,
    };
  }
}

/**
 * Network error for connection failures
 */
export class NetworkError extends Error {
  public readonly isNetworkError = true;
  public readonly timestamp: string;
  public readonly originalError?: Error;

  constructor(message: string, originalError?: Error) {
    super(message);
    this.name = 'NetworkError';
    this.timestamp = new Date().toISOString();
    this.originalError = originalError;

    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, NetworkError);
    }
  }

  static isNetworkError(error: unknown): error is NetworkError {
    return (
      error instanceof NetworkError ||
      (typeof error === 'object' &&
        error !== null &&
        'isNetworkError' in error &&
        (error as { isNetworkError: boolean }).isNetworkError === true)
    );
  }
}

/**
 * Timeout error for request timeouts
 */
export class TimeoutError extends Error {
  public readonly isTimeoutError = true;
  public readonly timestamp: string;
  public readonly timeoutMs: number;

  constructor(message: string, timeoutMs: number) {
    super(message);
    this.name = 'TimeoutError';
    this.timestamp = new Date().toISOString();
    this.timeoutMs = timeoutMs;

    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, TimeoutError);
    }
  }

  static isTimeoutError(error: unknown): error is TimeoutError {
    return (
      error instanceof TimeoutError ||
      (typeof error === 'object' &&
        error !== null &&
        'isTimeoutError' in error &&
        (error as { isTimeoutError: boolean }).isTimeoutError === true)
    );
  }
}

/**
 * Request interceptor type
 */
export type RequestInterceptor = (
  config: RequestConfig
) => RequestConfig | Promise<RequestConfig>;

/**
 * Response interceptor type
 */
export type ResponseInterceptor<T = unknown> = (
  response: ApiResponse<T>
) => ApiResponse<T> | Promise<ApiResponse<T>>;

/**
 * Error interceptor type
 */
export type ErrorInterceptor = (
  error: ApiError | NetworkError | TimeoutError
) => ApiError | NetworkError | TimeoutError | Promise<ApiError | NetworkError | TimeoutError>;

/**
 * Request configuration
 */
export interface RequestConfig {
  url: string;
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  signal?: AbortSignal;
  retries?: number;
  retryDelay?: number;
}

/**
 * API client configuration
 */
export interface ApiClientConfig {
  baseUrl: string;
  defaultTimeout?: number;
  defaultHeaders?: Record<string, string>;
  requestInterceptors?: RequestInterceptor[];
  responseInterceptors?: ResponseInterceptor[];
  errorInterceptors?: ErrorInterceptor[];
}